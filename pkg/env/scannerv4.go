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

	// ScannerV4AnonymousAuth specifies if Scanner V4 should authorize anonymous users. This is meant for debugging purposes.
	// Default: Enabled for non-release builds. Disabled for release builds.
	ScannerV4AnonymousAuth = RegisterBooleanSetting("ROX_SCANNER_V4_ALLOW_ANONYMOUS_AUTH", !buildinfo.ReleaseBuild)

	// ScannerV4ManifestGCInterval specifies the interval between manifest garbage collection runs.
	// The manifest garbage collector runs periodically to check for expired manifests and then delete a subset of them.
	ScannerV4ManifestGCInterval = registerDurationSetting("ROX_SCANNER_V4_MANIFEST_GC_INTERVAL", 4*time.Hour)

	// ScannerV4ManifestGCThrottle specifies the number of manifests to garbage collect during a typical, non-full run.
	ScannerV4ManifestGCThrottle = RegisterIntegerSetting("ROX_SCANNER_V4_MANIFEST_GC_THROTTLE", 100)

	// ScannerV4FullManifestGCInterval specifies the interval between full manifest garbage collection runs.
	// The manifest garbage collector runs periodically to check for expired manifests and then delete all of them.
	ScannerV4FullManifestGCInterval = registerDurationSetting("ROX_SCANNER_V4_FULL_MANIFEST_GC_INTERVAL", 24*time.Hour)

	// ScannerV4ManifestDeleteStart specifies the start of the interval in which manifests will be deleted.
	// Default: 7 days
	ScannerV4ManifestDeleteStart = registerDurationSetting("ROX_SCANNER_V4_MANIFEST_DELETE_INTERVAL_START", 7*24*time.Hour)

	// ScannerV4ManifestDeleteDuration specifies the duration of the interval (not inclusive) in which manifests will be deleted.
	// Default: 23 days
	ScannerV4ManifestDeleteDuration = registerDurationSetting("ROX_SCANNER_V4_MANIFEST_DELETE_DURATION", 23*24*time.Hour)
)
