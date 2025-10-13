package env

import "time"

// APITokenInvalidRetentionTime is the retention time for expired API tokens.
var APITokenInvalidRetentionTime = registerDurationSetting("ROX_TOKEN_INVALID_RETENTION_TIME", 24*time.Hour)
