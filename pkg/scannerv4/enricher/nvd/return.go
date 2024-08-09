package nvd

const (
	// Name of the enricher
	Name = `nvd`
	// Type is the type of data returned from the Enricher's Enrich method.
	Type = `message/vnd.stackrox.scannerv4.vulnerability; enricher=` + Name + ` schema=https://csrc.nist.gov/schema/nvd/api/2.0/source_api_json_2.0.schema`
)
