package env

var (
	// MaxMsgSizeSetting is the setting used for gRPC servers and clients to set maximum receive sizes.
	MaxMsgSizeSetting = RegisterIntegerSetting("ROX_GRPC_MAX_MESSAGE_SIZE", 24*1024*1024)
)
