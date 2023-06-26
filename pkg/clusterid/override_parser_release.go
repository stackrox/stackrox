//go:build release

package clusterid

// OverrideClusterIDParser does not do anything in release builds
func OverrideClusterIDParser(_ Parser) {
	log.Warn("Override clusterID parser must not be called in production code")
}
