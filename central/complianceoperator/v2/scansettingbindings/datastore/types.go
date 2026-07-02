package datastore

// DiscoveredScanConfig represents a scan configuration discovered from
// observed ScanSettingBindings in secured clusters. SSBs with the same
// name across clusters are grouped into a single DiscoveredScanConfig.
type DiscoveredScanConfig struct {
	Name         string
	ClusterIDs   []string
	ProfileNames []string
}
