package vsock

import (
	"strconv"

	"github.com/stackrox/rox/pkg/env"
)

var (
	// EnvVsockPort allows overriding the vsock port used by the compliance relay.
	// If unset or invalid, DefaultVsockPort is used by the relay layer.
	EnvVsockPort = env.RegisterSetting("ROX_COMPLIANCE_VSOCK_PORT", env.WithDefault(strconv.FormatUint(uint64(1234), 10)))
)
