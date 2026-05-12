package views

import "github.com/stackrox/rox/generated/storage"

// ContainerImageView represents a container image with its associated cluster IDs.
// This view is used to get distinct container images from active deployments
// along with the clusters where they are deployed.
type ContainerImageView struct {
	ImageIDV2         string   `db:"image_id"`
	ImageDigest       string   `db:"image_sha"`
	ClusterIDs        []string `db:"cluster_id"`
	ImageNameRegistry string   `db:"image_registry"`
	ImageNameRemote   string   `db:"image_remote"`
	ImageNameTag      string   `db:"image_tag"`
	ImageNameFullName string   `db:"image"`
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

// GetImageName returns the image name.
func (c *ContainerImageView) GetImageName() *storage.ImageName {
	return &storage.ImageName{
		Registry: c.ImageNameRegistry,
		Remote:   c.ImageNameRemote,
		Tag:      c.ImageNameTag,
		FullName: c.ImageNameFullName,
	}
}
