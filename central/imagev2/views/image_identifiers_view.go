package views

// ImageIdentifiersView contains essential identifying fields of an image:
// its V2 ID, digest, and full name.
type ImageIdentifiersView struct {
	ImageID  string `db:"image_id"`
	Digest   string `db:"image_sha"`
	FullName string `db:"image"`
}
