//go:build !release

package clusterid

// OverrideClusterIDParser the clusterID Parser. This should only be used for testing.
func OverrideClusterIDParser(parser Parser) {
	mu.Lock()
	defer mu.Unlock()
	instance = parser
}
