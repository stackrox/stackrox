package backend

import "github.com/stackrox/rox/pkg/env"

// These constants are used in the signed JWTs Central produces.
// const (
//
//	defaultTTL = 365 * 24 * time.Hour
//
// )
var (
	defaultTTL = env.APITokenValidityDuration.DurationSetting()
)
