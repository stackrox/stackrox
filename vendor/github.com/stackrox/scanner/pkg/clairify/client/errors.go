package client

import "errors"

var (
	// ErrorScanNotFound allows for external libraries to act upon the lack of a scan
	ErrorScanNotFound = errors.New("error scan not found")
)
