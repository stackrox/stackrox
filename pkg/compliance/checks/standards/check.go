package standards

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"k8s.io/kubelet/config/v1beta1"
)

// Check functions take a set of data about this compliance pod, perform a check, and return the results of that check
type Check func(complianceData *ComplianceData) []*storage.ComplianceResultValue_Evidence

// KubeletConfiguration wraps a typical v1beta1 configuration to provide access to the hostname-override flag in the Kubelet
type KubeletConfiguration struct {
	*v1beta1.KubeletConfiguration
	HostnameOverride string
}

// ComplianceData is the set of information we collect about this compliance pod
type ComplianceData struct {
	NodeName             string
	ScrapeID             string
	CommandLines         map[string]*compliance.CommandLine
	Files                map[string]*compliance.File
	SystemdFiles         map[string]*compliance.File
	ContainerRuntimeInfo *compliance.ContainerRuntimeInfo

	KubeletConfiguration *KubeletConfiguration

	Time         *types.Timestamp
	IsMasterNode bool
}
