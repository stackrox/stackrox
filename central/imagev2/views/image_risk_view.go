package views

type ImageV2RiskView struct {
	ImageID        string  `db:"image_id"`
	ImageRiskScore float32 `db:"image_risk_score"`
}
