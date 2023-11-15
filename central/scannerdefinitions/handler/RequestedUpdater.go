package handler

import (
	"os"
	"time"
)

// RequestedUpdater defines the methods for updating files.
type RequestedUpdater interface {
	Start()
	Stop()
	OpenFile() (*os.File, time.Time, error)
}
