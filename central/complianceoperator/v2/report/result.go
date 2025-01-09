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

// Results struct which holds the results of a report.
type Results struct {
	ResultCSVs map[string][]*ResultRow // map of cluster id to slice of *resultRow
	TotalPass  int
	TotalFail  int
	TotalMixed int
	Profiles   []string
	Clusters   int
}
