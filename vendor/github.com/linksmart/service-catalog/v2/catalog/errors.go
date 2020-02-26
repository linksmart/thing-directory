// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

// Not Found
type NotFoundError struct{ Msg string }

func (e *NotFoundError) Error() string { return e.Msg }

// Conflict (non-unique id, assignment to read-only data)
type ConflictError struct{ Msg string }

func (e *ConflictError) Error() string { return e.Msg }

// Bad Request
type BadRequestError struct{ Msg string }

func (e *BadRequestError) Error() string { return e.Msg }
