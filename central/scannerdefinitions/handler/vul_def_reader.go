package handler

import "io"

// vulDefReader unifies online and offline reader for vuln definitions
type vulDefReader interface {
	io.Reader
	io.Closer
	io.Seeker
	Name() string
}
