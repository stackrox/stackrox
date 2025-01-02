package report

// ResultRow struct which hold all columns of a report row
type ResultRow struct {
	ClusterName  string
	CheckName    string
	Profile      string
	ControlRef   string
	Description  string
	Status       string
	Remediation  string
	Rationale    string
	Instructions string
}
