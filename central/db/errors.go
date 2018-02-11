package db

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// An ErrNotFound indicates that the desired object could not be located.
type ErrNotFound struct {
	Type string
	ID   string
}

func (e ErrNotFound) Error() string {
	if e.Type != "" {
		return fmt.Sprintf("%s '%s' not found", e.Type, e.ID)
	}
	return fmt.Sprintf("'%s' not found", e.ID)
}

// Status implements the StatusError interface.
func (e ErrNotFound) Status() codes.Code {
	return codes.NotFound
}
