package views

type ImageIDAndDigestView struct {
	ImageID string `db:"image_id"`
	Digest  string `db:"image_sha"`
}
