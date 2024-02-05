package networkgraph

const (
	// InternetExternalSourceID is UUID for network nodes external to a cluster which are not identified by CIDR block or IP address.
	InternetExternalSourceID = "afa12424-bde3-4313-b810-bb463cbe8f90"
	// InternalSourceID is UUID for network nodes internal to a cluster which cannot be mapped to a deployment
	InternalSourceID = "ada12424-bde3-4313-b810-bb463cbe8f90"
	// InternetExternalSourceName is name for the Internet network node
	InternetExternalSourceName = "External Entities"
	// InternalEntitiesName is name for the internal-unknown network node
	InternalEntitiesName = "Internal Entities"
)

// IsConstantID returns true if entity ID matches one of the two hardcoded IDs
func IsConstantID(id string) bool {
	return id == InternetExternalSourceID || id == InternalSourceID
}
