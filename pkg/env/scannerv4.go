package env

import (
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/size"
)

var (
	// ScannerV4MaxRespMsgSize sets the maximum response size (in bytes) a Scanner v4 client may receive.
	// ROX_GRPC_MAX_MESSAGE_SIZE is the related server-side configuration.
	ScannerV4MaxRespMsgSize = RegisterIntegerSetting("ROX_SCANNER_V4_GRPC_MAX_RESPONSE_SIZE", 12*size.MB)

	// ScannerV4NodeJSSupport specifies if Scanner v4 should support indexing/vuln matching NodeJS (npm) packages.
	// TODO(ROX-21768): Support another alternative: show only NodeJS packages affected by fixable vulns (like Scanner v2).
	ScannerV4NodeJSSupport = RegisterBooleanSetting("ROX_SCANNER_V4_NODE_JS_SUPPORT", true)

	// ScannerV4AnonymousAuth specifies if Scanner V4 should authorize anonymous users. This is meant for debugging purposes.
	// Default: Enabled for non-release builds. Disabled for release builds.
	ScannerV4AnonymousAuth = RegisterBooleanSetting("ROX_SCANNER_V4_ALLOW_ANONYMOUS_AUTH", !buildinfo.ReleaseBuild)
)
