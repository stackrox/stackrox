package auth

import (
	"google.golang.org/grpc/credentials"
)

// Method describes the method of authentication.
type Method interface {
	Type() string
	GetCredentials(url string) (credentials.PerRPCCredentials, error)
}
