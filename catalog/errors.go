// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// Not Found
type NotFoundError struct{ s string }

func (e *NotFoundError) Error() string { return e.s }

// Conflict (non-unique id, assignment to read-only data)
type ConflictError struct{ s string }

func (e *ConflictError) Error() string { return e.s }

// Bad Request
type BadRequestError struct{ s string }

func (e *BadRequestError) Error() string { return e.s }

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
	if code >= 500 {
		log.Println("ERROR:", msg)
	}
	b, err := json.Marshal(e)
	if err != nil {
		log.Printf("ERROR serializing error object: %s", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(b)
	if err != nil {
		log.Printf("ERROR writing HTTP response: %s", err)
	}
}
