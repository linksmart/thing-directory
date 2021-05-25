// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/linksmart/thing-directory/wot"
	uuid "github.com/satori/go.uuid"
)

// Not Found
type NotFoundError struct{ S string }

func (e *NotFoundError) Error() string { return e.S }

// Conflict (non-unique id, assignment to read-only data)
type ConflictError struct{ S string }

func (e *ConflictError) Error() string { return e.S }

// Bad Request
type BadRequestError struct{ S string }

func (e *BadRequestError) Error() string { return e.S }

// Validation error (HTTP Bad Request)
type ValidationError struct {
	ValidationErrors []wot.ValidationError
}

func (e *ValidationError) Error() string { return "validation errors" }

// ErrorResponse writes error to HTTP ResponseWriter
func ErrorResponse(w http.ResponseWriter, code int, msg ...interface{}) {
	ProblemDetailsResponse(w, wot.ProblemDetails{
		Status: code,
		Detail: fmt.Sprint(msg...),
	})
}

func ValidationErrorResponse(w http.ResponseWriter, validationIssues []wot.ValidationError) {
	ProblemDetailsResponse(w, wot.ProblemDetails{
		Status:           http.StatusBadRequest,
		Detail:           "The input did not pass the JSON Schema validation",
		ValidationErrors: validationIssues,
	})
}

// ErrorResponse writes error to HTTP ResponseWriter
func ProblemDetailsResponse(w http.ResponseWriter, pd wot.ProblemDetails) {
	if pd.Title == "" {
		pd.Title = http.StatusText(pd.Status)
		if pd.Title == "" {
			panic(fmt.Sprint("Invalid HTTP status code: ", pd.Status))
		}
	}
	pd.Instance = "/errors/" + uuid.NewV4().String()
	log.Println("Problem Details instance:", pd.Instance)
	if pd.Status >= 500 {
		log.Println("ERROR:", pd.Detail)
	}
	b, err := json.Marshal(pd)
	if err != nil {
		log.Printf("ERROR serializing error object: %s", err)
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(pd.Status)
	_, err = w.Write(b)
	if err != nil {
		log.Printf("ERROR writing HTTP response: %s", err)
	}
}
