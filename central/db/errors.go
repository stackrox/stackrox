package db

import (
	"fmt"
)

// An ErrNotFound indicates that the desired object could not be located.
type ErrNotFound struct {
	ID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("'%s' not found", e.ID)
}
