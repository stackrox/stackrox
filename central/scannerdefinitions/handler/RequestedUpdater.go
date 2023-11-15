package handler

import (
	"os"
	"time"
)

// RequestedUpdater Interface defining methods for updating the data necessary for scanning
type RequestedUpdater interface {
	Start()
	Stop()
	OpenFile() (*os.File, time.Time, error)
}
