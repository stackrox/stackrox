package env

import "time"

var (
	// MaxConnectionAgeSetting configures gRPC server MaxConnectionAge for connection rebalancing in HA deployments.
	MaxConnectionAgeSetting = registerDurationSetting("ROX_GRPC_MAX_CONNECTION_AGE", 5*time.Minute)
)
