package env

import "github.com/stackrox/rox/pkg/size"

var (
	// ScannerV4MaxRespMsgSize sets the maximum response size (in bytes) a Scanner v4 client may receive.
	// ROX_GRPC_MAX_MESSAGE_SIZE is the related server-side configuration.
	ScannerV4MaxRespMsgSize = RegisterIntegerSetting("ROX_SCANNER_V4_GRPC_MAX_RESPONSE_SIZE", 12*size.MB)

	ScannerV4NodeJSSupport = RegisterBooleanSetting("ROX_SCANNER_V4_NODE_JS_SUPPORT", true)
)
