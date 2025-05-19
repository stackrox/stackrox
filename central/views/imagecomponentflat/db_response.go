package imagecomponentflat

type imageComponentFlatResponse struct {
	Component       string   `db:"component"`
	ComponentIDs    []string `db:"component_id"`
	Version         string   `db:"component_version"`
	TopCVSS         *float32 `db:"component_top_cvss_max"`
	Priority        *int64   `db:"component_risk_score_min"`
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

func (c *imageComponentFlatResponse) GetPriority() int64 {
	if c.Priority == nil {
		return 0
	}
	return *c.Priority
}

func (c *imageComponentFlatResponse) GetOperatingSystem() string {
	return c.OperatingSystem
}

type imageComponentFlatCount struct {
	ComponentCount int `db:"component_count"`
}
