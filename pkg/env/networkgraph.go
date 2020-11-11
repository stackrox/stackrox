package env

import "time"

var (
	// ExtNetworkSrcsGatherInterval is the frequency at which default external sources are gathered.
	ExtNetworkSrcsGatherInterval = registerDurationSetting("ROX_EXT_NETWORK_SRCS_GATHER_INTERVAL", time.Hour*24*7)
)
