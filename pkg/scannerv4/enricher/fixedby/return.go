package fixedby

const (
	// Name of the enricher
	Name = "fixedby"
	// Type is the type of data returned from the Enricher's Enrich method.
	Type = "message/vnd.stackrox.scannerv4.fixedby; enricher=" + Name
)
