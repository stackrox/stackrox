package errox

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
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

func joinIfNotEmpty(args ...string) string {
	sb := strings.Builder{}
	for _, arg := range args {
		if arg != "" {
			if sb.Len() != 0 {
				sb.WriteString(": ")
			}
			sb.WriteString(arg)
		}
	}
	return sb.String()
}

func isNilPointer(i any) bool {
	value := reflect.ValueOf(i)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func GetUserMessage(err error) string {
	for ; err != nil && !isNilPointer(err); err = errors.Unwrap(err) {
		switch e := err.(type) {
		case *userMessage:
			return joinIfNotEmpty(e.message, GetUserMessage(e.base))
		case *RoxError:
			if e.base != nil {
				return GetUserMessage(e.base)
			}
			return e.message
		case *errorhelpers.ErrorList:
			if e.ToError() == nil {
				return ""
			}
			messages := make([]string, 0, len(e.Errors()))
			for _, err := range e.Errors() {
				if message := GetUserMessage(err); message != "" {
					messages = append(messages, message)
				}
			}
			switch len(messages) {
			case 0:
				return e.StartString()
			case 1:
				return e.StartString() + ": " + messages[0]
			default:
				return e.StartString() + ": [" + strings.Join(messages, ", ") + "]"
			}
		case *multierror.Error:
			if e.Len() == 0 {
				return ""
			}
			messages := make([]string, 0, e.Len())
			for _, err := range e.Errors {
				if message := GetUserMessage(err); message != "" {
					messages = append(messages, message)
				}
			}
			switch len(messages) {
			case 0:
				return ""
			case 1:
				return messages[0]
			default:
				return fmt.Sprintf("[%s]", strings.Join(messages, ", "))
			}
		case *net.OpError:
			op := e.Op
			if e.Net != "" {
				op += " " + e.Net
			}
			return joinIfNotEmpty(op, GetUserMessage(e.Err))
		case *registry.HttpStatusError:
			if e.Response != nil {
				return "HTTP response: " + e.Response.Status
			}
			return "HTTP error"
		case *net.AddrError:
			return joinIfNotEmpty("address", e.Err)
		case *net.DNSError:
			return joinIfNotEmpty("lookup", e.Err)
		case *os.SyscallError:
			return joinIfNotEmpty(e.Syscall, GetUserMessage(e.Err))
		case *os.PathError:
			return joinIfNotEmpty(e.Op, GetUserMessage(e.Err))
		case *url.Error:
			return joinIfNotEmpty(e.Op, GetUserMessage(e.Err))
		case *strconv.NumError:
			return joinIfNotEmpty("error parsing "+strconv.Quote(e.Num), GetUserMessage(e.Err))
		default:
			continue
		}
	}
	return ""
}

func WithUserMessage(err error, format string, args ...any) error {
	return &userMessage{
		message: fmt.Sprintf(format, args...),
		base:    err,
	}
}
