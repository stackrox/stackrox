package handler

import "io"

// definitionFileReader unifies online and offline reader for vuln definitions
type definitionFileReader interface {
	io.Reader
	io.Closer
	io.Seeker
	Name() string
}
