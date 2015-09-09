package validator

import "net/http"

// Interface methods to validate Service Token
type Validator interface {
	// Given a valid serviceToken for the specified serviceID,
	//	ValidateTicket must return true with a set of user attributes.
	Validate(serviceToken string) (bool, map[string]string, error)
	// An HTTP handler wrapped around ValidateTicket
	//	which resonds based on the X_auth_token entity header
	Handler(next http.Handler) http.Handler
}
