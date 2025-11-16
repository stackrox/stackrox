// Package notaffected defines constants and structs related to Scanner V4's Not Affected enricher
// which are used across the pkg/ and scanner/ directories.
package notaffected

const (
	// Name is the name of the Not Affected enricher.
	Name = "stackrox.rhel-not-affected"

	// Type is the type of data returned from the CSAF Enricher's Enrich method.
	Type = `message/vnd.stackrox.scannerv4.map.notaffected; enricher=` + Name

	RedHatProducts = "red_hat_products"
)
