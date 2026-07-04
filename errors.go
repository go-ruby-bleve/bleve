package bleve

import (
	"errors"
	"fmt"
)

// Sentinel errors at the root of the Bleve::Error tree. Callers may test for
// them with errors.Is; every wrapped Error unwraps to one of these (or to the
// underlying bleve error).
var (
	// ErrClosed is returned when an operation is attempted on a closed index.
	ErrClosed = errors.New("bleve: index is closed")
	// ErrNotFound is returned when a document id is not present in the index.
	ErrNotFound = errors.New("bleve: document not found")
)

// Error is the common error type returned by this package. It records the
// operation that failed and wraps the underlying cause, which may be a sentinel
// (ErrClosed, ErrNotFound) or an error from the underlying bleve engine.
type Error struct {
	// Op is the high-level operation that failed, e.g. "index" or "search".
	Op string
	// Err is the wrapped cause.
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Op == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("bleve: %s: %v", e.Op, e.Err)
}

// Unwrap exposes the wrapped cause so errors.Is / errors.As work.
func (e *Error) Unwrap() error { return e.Err }

// wrap builds an *Error for op unless err is nil.
func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Op: op, Err: err}
}
