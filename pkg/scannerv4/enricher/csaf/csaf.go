// Package csaf defines constants and structs related to Scanner V4's CSAF enricher
// which are used across the pkg/ and scanner/ directories.
package csaf

import "time"

const (
	// Name is the name of the CSAF enricher.
	Name = "stackrox.rhel-csaf"

	// Type is the type of data returned from the CSAF Enricher's Enrich method.
	Type = `message/vnd.stackrox.scannerv4.map.csaf; enricher=` + Name
)

// Advisory represents a CSAF enrichment advisory record.
// It tracks attributes which should be consistent each time the advisory is output.
type Advisory struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ReleaseDate time.Time `json:"release_date"`
	Severity    string    `json:"severity"`
	CVSSv3      CVSS      `json:"cvssv3"`
	CVSSv2      CVSS      `json:"cvssv2"`
}

// CVSS represents CVSS metrics we care to track for the advisory.
type CVSS struct {
	Score  float32 `json:"score"`
	Vector string  `json:"vector"`
}
