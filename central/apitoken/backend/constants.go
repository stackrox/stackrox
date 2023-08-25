package backend

import (
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
)

// These constants are used in the signed JWTs Central produces.
const (
	defaultTTL = 365 * timeutil.HoursInDay * time.Hour
)
