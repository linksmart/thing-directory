// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	_ "github.com/linksmart/go-sec/auth/keycloak/obtainer"
)

// Handler is a http.Handler that validates tickets and performs optional authorization
func (v *Validator) Handler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// Authorization header
		Authorization := r.Header.Get("Authorization")
		if Authorization == "" {
			if v.authz != nil {
				if ok := v.authz.Rules.Authorized(r.URL.Path, r.Method, nil); ok {
					// Anonymous access, proceed to the next handler
					next.ServeHTTP(w, r)
					return
				}
			}
			errorResponse(w, http.StatusUnauthorized, "unauthorized request.")
			return
		}

		parts := strings.SplitN(Authorization, " ", 2)
		if len(parts) != 2 {
			errorResponse(w, http.StatusBadRequest, "invalid format for Authorization header value")
			return
		}
		method, value := parts[0], parts[1]

		switch {
		case method == "Bearer": // i.e. Authorization: Bearer token
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
			errorResponse(w, http.StatusUnauthorized, "unsupported Authorization method: "+method)
			return
		}

		// Successful validation, proceed to the next handler
		next.ServeHTTP(w, r)
		return
	}
	return http.HandlerFunc(fn)
}

// validationChain validates a token and performs authorization
func (v *Validator) validationChain(tokenString string, path, method string) (int, error) {
	// Validate Token
	valid, claims, err := v.driver.Validate(v.serverAddr, v.clientID, tokenString)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("validation error: %s", err)
	}
	if !valid {
		if claims != nil && claims.Status != "" {
			return http.StatusUnauthorized, fmt.Errorf("unauthorized request: %s", claims.Status)
		}
		return http.StatusUnauthorized, fmt.Errorf("unauthorized request")
	}
	// Check for optional authorization
	if v.authz.Enabled {
		if ok := v.authz.Rules.Authorized(path, method, claims); !ok {
			return http.StatusForbidden, fmt.Errorf("access forbidden")
		}
	}
	return http.StatusOK, nil
}

// errorResponse writes error to HTTP ResponseWriter
func errorResponse(w http.ResponseWriter, code int, message string) {
	b, _ := json.Marshal(map[string]interface{}{
		"code":    code,
		"message": message,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)
}
