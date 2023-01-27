package backend

import "time"

// These constants are used in the signed JWTs Central produces.
const (
	// defaultTTL = 365 * 24 * time.Hour
	defaultTTL = 1 * time.Hour
)
