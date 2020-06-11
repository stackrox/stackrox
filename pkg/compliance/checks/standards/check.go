package standards

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	internalTypes "github.com/stackrox/rox/pkg/docker/types"
)

// Check functions take a set of data about this compliance pod, perform a check, and return the results of that check
type Check func(complianceData *ComplianceData) []*storage.ComplianceResultValue_Evidence

// ComplianceData is the set of information we collect about this compliance pod
type ComplianceData struct {
	NodeName             string
	ScrapeID             string
	DockerData           *internalTypes.Data
	CommandLines         map[string]*compliance.CommandLine
	Files                map[string]*compliance.File
	SystemdFiles         map[string]*compliance.File
	ContainerRuntimeInfo *compliance.ContainerRuntimeInfo
	Time                 *types.Timestamp
}
