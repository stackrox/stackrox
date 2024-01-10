package env

import "github.com/stackrox/rox/pkg/size"

var (
	ScannerV4MaxRespMsgSize = RegisterIntegerSetting("ROX_SCANNER_V4_GRPC_MAX_RESPONSE_SIZE", 64*size.MB)
)
