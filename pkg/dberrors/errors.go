package dberrors

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// An ErrNotFound indicates that the desired object could not be located.
type ErrNotFound struct {
	Type string
	ID   string
}

func (e ErrNotFound) Error() string {
	sb := strings.Builder{}
	if e.Type != "" {
		_, _ = sb.WriteString(fmt.Sprintf("%s ", e.Type))
	}
	if e.ID != "" {
		_, _ = sb.WriteString(fmt.Sprintf("'%s' ", e.ID))
	}
	_, _ = sb.WriteString("not found")
	return sb.String()
}

// GRPCStatus returns the error as a `status.Status` object. It is required to ensure interoperability with
// `status.FromError()`.
func (e ErrNotFound) GRPCStatus() *status.Status {
	return status.New(codes.NotFound, e.Error())
}
