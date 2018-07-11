package authproviders

import "time"

// A User is a human (not a machine) who uses the system.
type User struct {
	ID string
}

// An Authenticator knows how to parse API metadata (gRPC metadata,
// or HTTP headers) into a User identity.
type Authenticator interface {
	Enabled() bool
	Validated() bool
	User(headers map[string][]string) (user User, expiration time.Time, err error)
	LoginURL() string
	RefreshURL() string
}
