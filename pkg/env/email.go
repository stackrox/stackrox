package env

import "time"

var (
	// EmailConnectTimeout controls the timeout for establishing an SMTP connection,
	// including both the TCP dial and waiting for the server's 220 greeting banner.
	// SMTP servers may delay the banner for reverse DNS lookups or other checks,
	// which can exceed the default timeout.
	EmailConnectTimeout = registerDurationSetting("ROX_EMAIL_CONNECT_TIMEOUT", 5*time.Second)
)
