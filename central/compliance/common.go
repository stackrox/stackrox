package compliance

// NewPair returns a new ClusterStandardPair
func NewPair(clusterID, standardID string) ClusterStandardPair {
	return ClusterStandardPair{
		ClusterID:  clusterID,
		StandardID: standardID,
	}
}

// ClusterStandardPair is a (cluster ID, standard ID) combination.
type ClusterStandardPair struct {
	ClusterID, StandardID string
}
