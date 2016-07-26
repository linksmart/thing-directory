// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"linksmart.eu/lc/sec/auth"
	_ "linksmart.eu/lc/sec/auth/cas/obtainer"
	_ "linksmart.eu/lc/sec/auth/keycloak/obtainer"
	"linksmart.eu/lc/sec/auth/obtainer"
)

// Handler is a http.Handler that validates tickets and performs optional authorization
func (v *Validator) Handler(next http.Handler) http.Handler {
	//return v.driver.Handler(v.serverAddr, v.serviceID, v.authz, next)
	fn := func(w http.ResponseWriter, r *http.Request) {

		// X-Auth-Token header
		// Deprecated. Use Authorization field instead.
		token := r.Header.Get("X-Auth-Token")
		if token != "" {
			v.validateHandlerFunc(w, r, token, next)
			return
		}

		// Authorization header
		Authorization := r.Header.Get("Authorization")
		if Authorization == "" {
			logger.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Unauthorized request.")
			auth.HTTPErrorResponse(http.StatusUnauthorized, "Unauthorized request.", w)
			return
		}

		parts := strings.SplitN(Authorization, " ", 2)
		if len(parts) != 2 {
			logger.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Invalid format for Authorization header field.")
			auth.HTTPErrorResponse(http.StatusBadRequest, "Invalid format for Authorization header field.", w)
			return
		}
		method, value := parts[0], parts[1]

		switch method {
		case "Basic": // i.e. Authorization: Basic base64_encoded_credentials
			var (
				err        error
				statuscode int
			)

			token, statuscode, err = v.basicAuth(value)
			if err != nil {
				logger.Printf("[%s] %q %s\n", r.Method, r.URL.String(), err.Error())
				auth.HTTPErrorResponse(statuscode, err.Error(), w)
				return
			}
			v.validateHandlerFunc(w, r, token, next)
			return

		case "Bearer": // i.e. Authorization: Bearer token
			// value == token
			v.validateHandlerFunc(w, r, value, next)
			return

		default:
			logger.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Invalid Authorization method.")
			auth.HTTPErrorResponse(http.StatusUnauthorized, "Invalid Authorization method.", w)
			return
		}

	}
	return http.HandlerFunc(fn)
}

// validateHandlerFunc validates token and performs authorization
//	If they both pass, next handler will be served
func (v *Validator) validateHandlerFunc(w http.ResponseWriter, r *http.Request, token string, next http.Handler) {
	// Validate Token
	valid, profile, err := v.driver.Validate(v.serverAddr, v.serviceID, token)
	if err != nil {
		logger.Printf("[%s] %q %s\n", r.Method, r.URL.String(), "Authentication server error: "+err.Error())
		auth.HTTPErrorResponse(http.StatusInternalServerError, "Authentication server error: "+err.Error(), w)
		return
	}
	if !valid {
		if profile != nil && profile.Status != "" {
			logger.Printf("[%s] %q %s\n", r.Method, r.URL.String(), profile.Status)
			auth.HTTPErrorResponse(http.StatusUnauthorized, "Unauthorized request: "+profile.Status, w)
			return
		}
		auth.HTTPErrorResponse(http.StatusUnauthorized, "Unauthorized request.", w)
		return
	}

	// Check for optional authorization
	if v.authz != nil {
		// Check if user matches authorization rules
		authorized := v.authz.Authorized(r.URL.Path, r.Method, profile.Username, profile.Groups)
		if !authorized {
			logger.Printf("[%s] %q Access denied for user `%s` member of %s\n", r.Method, r.URL.String(),
				profile.Username, profile.Groups)
			auth.HTTPErrorResponse(http.StatusForbidden,
				fmt.Sprintf("Access denied for user `%s` member of %s", profile.Username, profile.Groups), w)
			return
		}
	}

	// Valid token, proceed to next handler
	next.ServeHTTP(w, r)
}

// Cached clients for Basic auth
var clients = make(map[string]*obtainer.Client)

// basicAuth generates a token for the given credentials
//	Tokens are cached and are only regenerated if no longer valid
func (v *Validator) basicAuth(credentials string) (string, int, error) {

	b, err := base64.StdEncoding.DecodeString(credentials)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("Basic Auth: Invalid value: %s", err)
	}

	client, found := clients[credentials]
	if !found {
		pair := strings.SplitN(string(b), ":", 2)
		if len(pair) != 2 {
			return "", http.StatusBadRequest, fmt.Errorf("Basic Auth: Invalid value: %s", string(b))
		}

		// Setup ticket client
		client, err = obtainer.NewClient(v.driverName, v.serverAddr, pair[0], pair[1], v.serviceID)
		if err != nil {
			return "", http.StatusInternalServerError, fmt.Errorf("Basic Auth: Unable to create client for token generation: %s", err)

		}

		clients[credentials] = client
	}

	token, err := client.Obtain()
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("Basic Auth: Unable to obtain ticket: %s", err)

	}

	valid, _, err := v.driver.Validate(v.serverAddr, v.serviceID, token)
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("Basic Auth: Validation error: %s", err)

	}
	if !valid {
		token, err = client.Renew()
		if err != nil {
			return "", http.StatusUnauthorized, fmt.Errorf("Basic Auth: Unable to renew ticket: %s", err)

		}
	}
	return token, http.StatusOK, nil
}
