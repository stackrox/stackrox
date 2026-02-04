package views

// ContainerImageView represents a container image with its associated cluster IDs.
// This view is used to get distinct container images from active deployments
// along with the clusters where they are deployed.
type ContainerImageView struct {
	ImageIDV2 string `db:"image_id"`
	// Each imageIDV2 will map to a single digest, but SQL wants us to apply an aggreate func to all selected fields
	// when grouing by imageIDV2. So we need to select the digest with a min aggregate,
	// which will be a no-op for a single value.
	ImageDigest string   `db:"image_sha"`
	ClusterIDs  []string `db:"cluster_id"`
}

// GetImageDigest returns the image digest.
func (c *ContainerImageView) GetImageDigest() string {
	return c.ImageDigest
}

// GetImageID returns the V2 image ID.
func (c *ContainerImageView) GetImageID() string {
	return c.ImageIDV2
}

// GetClusterIDs returns the cluster IDs.
func (c *ContainerImageView) GetClusterIDs() []string {
	return c.ClusterIDs
}
