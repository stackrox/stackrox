package baseimage

import "time"

// BaseImage represents a row in the base_images table.
type BaseImage struct {
	ID           int64     `db:"id"`
	Registry     string    `db:"registry"`
	Repository   string    `db:"repository"`
	Tag          *string   `db:"tag"`           // Use pointer for nullable columns
	Digest       *string   `db:"digest"`        // Use pointer for nullable columns
	ConfigDigest *string   `db:"config_digest"` // Use pointer for nullable columns
	CreatedAt    time.Time `db:"created_at"`
	Active       *bool     `db:"active"` // Use pointer for nullable boolean
}

// BaseImageLayer represents a row in the base_image_layer table.
type BaseImageLayer struct {
	ID        int64  `db:"id"`
	IID       int64  `db:"iid"` // Foreign key to base_images.id
	LayerHash string `db:"layer_hash"`
	Level     int    `db:"level"`
}

// AddBaseImageInput represents the data needed to add a new base image and its layers.
// This struct will be passed as a parameter to the AddBaseImage method.
type AddBaseImageInput struct {
	BaseImage BaseImage
	Layers    []BaseImageLayer
}

// ImageMeta lets you hang useful metadata at a node that ends an image.
type ImageMeta struct {
	ImageID string   // e.g., manifest digest or internal id
	Tags    []string // optional: repo:tag list
}

type Match struct {
	Depth       int
	Node        *Node
	MatchedPath []string // normalized digests up to Depth
	Images      []ImageMeta
}
