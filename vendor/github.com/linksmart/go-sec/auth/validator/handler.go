// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	_ "github.com/linksmart/go-sec/auth/keycloak/obtainer"
	"github.com/linksmart/go-sec/auth/obtainer"
)

// Handler is a http.Handler that validates tickets and performs optional authorization
func (v *Validator) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// X-Auth-Token header
		// DEPRECATED: Use Authorization field instead.
		token := r.Header.Get("X-Auth-Token")
		if token != "" {
			statuscode, err := v.validationChain(token, r.URL.Path, r.Method)
			if err != nil {
				errorResponse(w, statuscode, err.Error())
				return
			}
			// Successful validation, proceed to the next handler
			next.ServeHTTP(w, r)
			return
		}

		// Authorization header
		Authorization := r.Header.Get("Authorization")
		if Authorization == "" {
			if v.authz != nil {
				if ok := v.authz.Authorized(r.URL.Path, r.Method, "", []string{"anonymous"}); ok {
					// Anonymous access, proceed to the next handler
					next.ServeHTTP(w, r)
					return
				}
			}
			errorResponse(w, http.StatusUnauthorized, "Unauthorized request.")
			return
		}

		parts := strings.SplitN(Authorization, " ", 2)
		if len(parts) != 2 {
			errorResponse(w, http.StatusBadRequest, "Invalid format for Authorization header field.")
			return
		}
		method, value := parts[0], parts[1]

		switch {
		case method == "Bearer": // i.e. Authorization: Bearer token
			// value == token
			statuscode, err := v.validationChain(value, r.URL.Path, r.Method)
			if err != nil {
				errorResponse(w, statuscode, err.Error())
				return
			}

		case method == "Basic" && v.basicEnabled: // i.e. Authorization: Basic base64_encoded_credentials
			token, statuscode, err := v.basicAuth(value)
			if err != nil {
				errorResponse(w, statuscode, err.Error())
				return
			}
			statuscode, err = v.validationChain(token, r.URL.Path, r.Method)
			if err != nil {
				errorResponse(w, statuscode, err.Error())
				return
			}

		default:
			errorResponse(w, http.StatusUnauthorized, "Unsupported Authorization method:", method)
			return
		}

		// Successful validation, proceed to the next handler
		next.ServeHTTP(w, r)
		return
	}
	return http.HandlerFunc(fn)
}

// validationChain validates a token and performs authorization
func (v *Validator) validationChain(token, path, method string) (int, error) {
	// Validate Token
	valid, profile, err := v.driver.Validate(v.serverAddr, v.clientID, token)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Authentication server error: %s", err)
	}
	if !valid {
		if profile != nil && profile.Status != "" {
			return http.StatusUnauthorized, fmt.Errorf("Unauthorized request: %s", profile.Status)
		}
		return http.StatusUnauthorized, fmt.Errorf("Unauthorized request")
	}
	// Check for optional authorization
	if v.authz != nil {
		if ok := v.authz.Authorized(path, method, profile.Username, profile.Groups); !ok {
			return http.StatusForbidden, fmt.Errorf("Access denied for user `%s` member of %s", profile.Username, profile.Groups)
		}
	}
	return http.StatusOK, nil
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
		client, err = obtainer.NewClient(v.driverName, v.serverAddr, pair[0], pair[1], v.clientID)
		if err != nil {
			return "", http.StatusInternalServerError, fmt.Errorf("Basic Auth: Unable to create client for token generation: %s", err)
		}

		clients[credentials] = client
	}

	token, err := client.Obtain()
	if err != nil {
		return "", http.StatusUnauthorized, fmt.Errorf("Basic Auth: Unable to obtain ticket: %s", err)
	}

	valid, _, err := v.driver.Validate(v.serverAddr, v.clientID, token)
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

// errorResponse writes error to HTTP ResponseWriter
func errorResponse(w http.ResponseWriter, code int, msgs ...string) {
	msg := strings.Join(msgs, " ")
	e := map[string]interface{}{
		"code":    code,
		"message": msg,
	}
	if code >= 500 {
		logger.Printf("ERROR %s: %s", http.StatusText(code), msg)
	} else {
		logger.Printf("%s: %s", http.StatusText(code), msg)
	}
	b, _ := json.Marshal(e)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}
