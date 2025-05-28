package imagecomponentflat

type imageComponentFlatResponse struct {
	Component       string   `db:"component"`
	ComponentIDs    []string `db:"component_id"`
	Version         string   `db:"component_version"`
	TopCVSS         *float32 `db:"component_top_cvss_max"`
	RiskScore       *float32 `db:"component_risk_priority_score_max"`
	OperatingSystem string   `db:"operating_system"`
}

func (c *imageComponentFlatResponse) GetComponent() string {
	return c.Component
}

func (c *imageComponentFlatResponse) GetComponentIDs() []string {
	return c.ComponentIDs
}

func (c *imageComponentFlatResponse) GetVersion() string {
	return c.Version
}

func (c *imageComponentFlatResponse) GetTopCVSS() float32 {
	if c.TopCVSS == nil {
		return 0.0
	}
	return *c.TopCVSS
}

func (c *imageComponentFlatResponse) GetRiskScore() float32 {
	if c.RiskScore == nil {
		return 0.0
	}
	return *c.RiskScore
}

func (c *imageComponentFlatResponse) GetOperatingSystem() string {
	return c.OperatingSystem
}
