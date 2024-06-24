package errox

import (
	"fmt"

	"github.com/pkg/errors"
)

type userMessage struct {
	message string
	base    error
}

func (u *userMessage) Unwrap() error {
	if u == nil {
		return nil
	}
	return u.base
}

func (u *userMessage) Error() string {
	if u == nil {
		return ""
	}
	if u.base != nil {
		return u.message + ": " + u.base.Error()
	}

	return u.message
}

func (u *userMessage) UserMessage() string {
	var um *userMessage
	if errors.As(u.base, &um) {
		return u.message + ": " + um.UserMessage()
	}
	return u.message
}

func WithUserMessage(err error, format string, args ...any) error {
	return &userMessage{
		message: fmt.Sprintf(format, args...),
		base:    err,
	}
}
