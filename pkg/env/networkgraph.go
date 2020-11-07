package env

import "time"

var (
	// NetworkGraphDefaultExtSrcsGatherFreq is the frequency at which default external sources are gathered.
	NetworkGraphDefaultExtSrcsGatherFreq = registerDurationSetting("ROX_NETWORK_GRAPH_DEFAULT_EXT_SRCS_GATHER_FREQUENCY", time.Hour*24*7)
)
