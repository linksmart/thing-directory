package resource

import (
	"net/http"
	"strings"
	"encoding/json"
)

// Not Found
type NotFoundError struct { s string }
func (e *NotFoundError) Error() string { return e.s }

// Not Unique
type NotUniqueError struct { s string }
func (e *NotUniqueError) Error() string { return e.s }

// Conflict (assignment to read-only data, etc)
type ConflictError struct { s string }
func (e *ConflictError) Error() string {	return e.s }



// Error describes an API error (serializable in JSON)
type Error struct {
	// Code is the (http) code of the error
	Code int `json:"code"`
	// Message is the (human-readable) error message
	Message string `json:"message"`
}

// ErrorResponse writes error to HTTP ResponseWriter
func ErrorResponse(w http.ResponseWriter, code int, msgs ...string) {
	msg := strings.Join(msgs, " ")
	e := &Error{
		code,
		msg,
	}
	logger.Println("ERROR:", msg)
	b, _ := json.Marshal(e)
	w.Header().Set("Content-Type", "application/json;version="+ApiVersion)
	w.WriteHeader(code)
	w.Write(b)
}