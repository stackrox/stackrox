package errox

import (
	"fmt"
	"net"

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
	if u == nil {
		return ""
	}
	var um *userMessage
	if errors.As(u.base, &um) {
		return u.message + ": " + um.UserMessage()
	}
	if extracted := extractUserMessage(u.base); extracted != "" {
		return u.message + ": " + extracted
	}
	return u.message
}

func extractUserMessage(err error) string {
	if e := (*net.AddrError)(nil); errors.As(err, &e) {
		return "address: " + e.Err
	}
	if e := (*net.DNSError)(nil); errors.As(err, &e) {
		return "lookup: " + e.Err
	}
	if e := (*net.OpError)(nil); errors.As(err, &e) {
		s := e.Op
		if e.Net != "" {
			s += " " + e.Net
		}
		return s + ": " + e.Err.Error()
	}
	return ""
}

func WithUserMessage(err error, format string, args ...any) error {
	return &userMessage{
		message: fmt.Sprintf(format, args...),
		base:    err,
	}
}
