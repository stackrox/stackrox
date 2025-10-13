// Package fixedby defines constants related to Scanner V4's FixedBy enricher
// which are used across the pkg/ and scanner/ directories.
package fixedby

const (
	// Name is the name of the FixedBy enricher.
	Name = "fixedby"
	// Type is the type of data returned from the FixedBy Enricher's Enrich method.
	Type = "message/vnd.stackrox.scannerv4.fixedby; enricher=" + Name
)
