package ratetracker

// ClusterRateTracker defines methods for tracking message rates in clusters.
type ClusterRateTracker interface {
	// ReceiveMsg records the receipt of a message from the specified cluster.
	ReceiveMsg(clusterID string)

	// IsTopCluster checks if the specified cluster is considered
	// a top-rated cluster.
	IsTopCluster(clusterID string) bool

	// Remove removes the tracking data for the specified cluster.
	Remove(clusterID string)
}
