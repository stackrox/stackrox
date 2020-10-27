package authproviders

import (
	"github.com/pkg/errors"
)

// CreateError logs the error along with the message string, and returns error
func CreateError(message string, err error) error {
	log.Errorf("Auth Provider Error: %s: %v", message, err)
	return errors.Wrap(err, message)
}
