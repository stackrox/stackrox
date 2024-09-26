package env

import (
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/size"
)

var (
	// ScannerV4MaxRespMsgSize sets the maximum response size (in bytes) a Scanner v4 client may receive.
	// ROX_GRPC_MAX_MESSAGE_SIZE is the related server-side configuration.
	ScannerV4MaxRespMsgSize = RegisterIntegerSetting("ROX_SCANNER_V4_GRPC_MAX_RESPONSE_SIZE", 12*size.MB)

	// ScannerV4PartialNodeJSSupport specifies if Scanner v4 should support partial indexing/vuln matching Node.js (npm) packages.
	// Partial support is equivalent to StackRox Scanner (Scanner v2) support: only return packages which are affected
	// by at least one vulnerability.
	ScannerV4PartialNodeJSSupport = RegisterBooleanSetting("ROX_SCANNER_V4_PARTIAL_NODE_JS_SUPPORT", false)

	// ScannerV4AnonymousAuth specifies if Scanner V4 should authorize anonymous users. This is meant for debugging purposes.
	// Default: Enabled for non-release builds. Disabled for release builds.
	ScannerV4AnonymousAuth = RegisterBooleanSetting("ROX_SCANNER_V4_ALLOW_ANONYMOUS_AUTH", !buildinfo.ReleaseBuild)

	// ScannerV4ManifestGCInterval specifies the interval between manifest garbage collection runs.
	// The manifest garbage collector runs periodically to check for expired manifests and then delete them.
	ScannerV4ManifestGCInterval = registerDurationSetting("ROX_SCANNER_V4_MANIFEST_GC_INTERVAL", 6*time.Hour)
)
