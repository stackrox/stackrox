package views

type ImageRiskView struct {
	ImageID        string  `db:"image_sha"`
	ImageRiskScore float32 `db:"image_risk_score"`
}
