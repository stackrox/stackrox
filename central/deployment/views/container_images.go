package views

// ContainerImagesResponse represents a container image with its associated cluster IDs.
// This view is used to get distinct container images from active deployments
// along with the clusters where they are deployed.
type ContainerImagesResponse struct {
	ImageIDV2  string   `db:"image_id"`
	ClusterIDs []string `db:"cluster_id"`
}

// GetImageID returns the V2 image ID.
func (c *ContainerImagesResponse) GetImageID() string {
	return c.ImageIDV2
}

// GetClusterIDs returns the cluster IDs.
func (c *ContainerImagesResponse) GetClusterIDs() []string {
	return c.ClusterIDs
}
