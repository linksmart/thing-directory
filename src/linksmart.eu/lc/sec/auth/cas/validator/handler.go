// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"linksmart.eu/lc/sec/auth"
	_ "linksmart.eu/lc/sec/auth/cas/obtainer"
	"linksmart.eu/lc/sec/auth/obtainer"
	"linksmart.eu/lc/sec/authz"
)

// Cached clients for Basic auth
var clients = make(map[string]*obtainer.Client)

// HTTP Handler for service ticket validation
func (v *CASValidator) Handler(serverAddr, serviceID string, authz *authz.Conf, next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		var (
			valid, validated bool
			profile          map[string]string
		)

		// X-Auth-Token is deprecated. Use Authorization field instead.
		token := r.Header.Get("X-Auth-Token")
		if token == "" {

			Authorization := r.Header.Get("Authorization")
			if Authorization == "" {
				auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Unauthorized request.")
				auth.HTTPErrorResponse(http.StatusUnauthorized, "Unauthorized request.", w)
				return
			}

			parts := strings.SplitN(Authorization, " ", 2)
			if len(parts) != 2 {
				auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Invalid format for Authorization header field.")
				auth.HTTPErrorResponse(http.StatusBadRequest, "Invalid format for Authorization header field.", w)
				return
			}
			method, value := parts[0], parts[1]

			switch method {
			case "Basic": // i.e. Authorization: Basic base64_encoded_credentials
				b, err := base64.StdEncoding.DecodeString(value)
				if err != nil {
					auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Basic Auth: Invalid value: "+err.Error())
					auth.HTTPErrorResponse(http.StatusBadRequest, "Basic Auth: Invalid value: "+err.Error(), w)
					return
				}

				client, found := clients[value]
				if !found {
					pair := strings.SplitN(string(b), ":", 2)
					if len(pair) != 2 {
						auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Basic Auth: Invalid value: "+string(b))
						auth.HTTPErrorResponse(http.StatusBadRequest, "Basic Auth: Invalid value.", w)
						return
					}

					// Setup ticket client
					client, err = obtainer.NewClient(driverName, serverAddr, pair[0], pair[1], serviceID)
					if err != nil {
						auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Basic Auth: Unable to create client for token generation.")
						auth.HTTPErrorResponse(http.StatusInternalServerError, "Basic Auth: Unable to create client for token generation.", w)
						return
					}

					clients[value] = client
				}

				token, err = client.Obtain()
				if err != nil {
					auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Basic Auth: "+err.Error())
					auth.HTTPErrorResponse(http.StatusUnauthorized, "Basic Auth: "+err.Error(), w)
					return
				}

				valid, profile, err = v.Validate(serverAddr, serviceID, token)
				if err != nil {
					auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Basic Auth: Validation error: "+err.Error())
					auth.HTTPErrorResponse(http.StatusInternalServerError, "Basic Auth: Validation error: "+err.Error(), w)
					return
				}
				if !valid {
					token, err = client.Renew()
					if err != nil {
						auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Basic Auth: "+err.Error())
						auth.HTTPErrorResponse(http.StatusUnauthorized, "Basic Auth: "+err.Error(), w)
						return
					}
				} else {
					validated = true
				}

			case "Bearer": // i.e. Authorization: Bearer token
				token = value

			default:
				auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Invalid Authorization method.")
				auth.HTTPErrorResponse(http.StatusUnauthorized, "Invalid Authorization method.", w)
				return
			}
		}

		// Validate Token
		if !validated {
			var err error
			valid, profile, err = v.Validate(serverAddr, serviceID, token)
			if err != nil {
				auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Authentication server error: "+err.Error())
				auth.HTTPErrorResponse(http.StatusInternalServerError, "Authentication server error: "+err.Error(), w)
				return
			}
			if !valid {
				if _, ok := profile["error"]; ok {
					auth.Log.Printf("[%s] %q %s\n", r.Method, r.URL.String(), profile["error"])
					auth.HTTPErrorResponse(http.StatusUnauthorized, "Unauthorized request: "+profile["error"], w)
					return
				}
				auth.HTTPErrorResponse(http.StatusUnauthorized, "Unauthorized request.", w)
				return
			}
		}

		// Check for optional authorization
		if authz != nil {
			// Check if user matches authorization rules
			authorized := authz.Authorized(r.URL.Path, r.Method, profile["user"], profile["group"])
			if !authorized {
				auth.Log.Printf("[%s] %q %s `%s`/`%s`\n", r.Method, r.URL.String(),
					"Access denied for", profile["group"], profile["user"])
				auth.HTTPErrorResponse(http.StatusForbidden,
					fmt.Sprintf("Access denied for `%s`/`%s`", profile["group"], profile["user"]), w)
				return
			}
		}

		// Valid token, proceed to next handler
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
