package constants

const (
	// FixedByName is the name of the FixedBy enricher
	FixedByName = "fixedby"
	// FixedByType is the type of data returned from the FixedBy Enricher's Enrich method.
	FixedByType = "message/vnd.stackrox.scannerv4.fixedby; enricher=" + FixedByName

	// NVDName is the name of the NVD enricher
	NVDName = `nvd`
	// NVDType is the type of data returned from the NVD Enricher's Enrich method.
	NVDType = `message/vnd.stackrox.scannerv4.vulnerability; enricher=` + NVDName + ` schema=https://csrc.nist.gov/schema/nvd/api/2.0/source_api_json_2.0.schema`

	// ManualUpdaterName is the name of the manual updater
	ManualUpdaterName = "stackrox-manual"
)
