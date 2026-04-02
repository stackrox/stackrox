package views

// ComponentRiskView is a lightweight view struct for initializing component rankers.
// It fetches only the component ID and risk score, avoiding full protobuf deserialization.
type ComponentRiskView struct {
	ComponentID        string  `db:"component_id"`
	ComponentRiskScore float32 `db:"component_risk_score"`
}
