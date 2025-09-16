// Package nvd defines constants related to Scanner V4's NVD enricher
// which are used across the pkg/ and scanner/ directories.
package nvd

const (
	// Name is the name of the NVD enricher.
	Name = `nvd`
	// Type is the type of data returned from the NVD Enricher's Enrich method.
	Type = `message/vnd.stackrox.scannerv4.vulnerability; enricher=` + Name + ` schema=https://csrc.nist.gov/schema/nvd/api/2.0/source_api_json_2.0.schema`
)
