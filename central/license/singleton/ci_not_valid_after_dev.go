// +build !release

package singleton

import (
	"time"
)

const (
	ciSigningKeyLatestNotValidAfterOffset = 365 * 24 * time.Hour
)
