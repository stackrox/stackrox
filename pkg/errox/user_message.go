package errox

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

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

func GetUserMessage(err error) string {
	for ; err != nil; err = errors.Unwrap(err) {
		switch e := err.(type) {
		case *userMessage:
			return joinIfNotEmpty(e.message, GetUserMessage(e.base))
		case *net.OpError:
			op := e.Op
			if e.Net != "" {
				op += " " + e.Net
			}
			return joinIfNotEmpty(op, GetUserMessage(e.Err))
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
