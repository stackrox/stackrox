package apitoken

import "time"

// These constants are used in the signed JWTs Central produces.
const (
	Issuer        = "central"
	Audience      = "central"
	DefaultExpiry = 365 * 24 * time.Hour
)
