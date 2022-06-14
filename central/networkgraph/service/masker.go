package service

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	// MaskedDeploymentName is name of a masked deployment
	MaskedDeploymentName = "masked deployment"
	// MaskedNamespaceName is name of a masked namespace
	MaskedNamespaceName = "masked namespace"
)

type flowGraphMasker struct {
	maskedDeployments      []*storage.ListDeployment
	realToMaskedDeployment map[string]*storage.ListDeployment
	realToMaskedNamespace  map[string]string
}

func newFlowGraphMasker() *flowGraphMasker {
	return &flowGraphMasker{
		realToMaskedDeployment: make(map[string]*storage.ListDeployment),
		realToMaskedNamespace:  make(map[string]string),
	}
}

func (m *flowGraphMasker) GetMaskedDeployment(origDeployment *storage.ListDeployment) *storage.ListDeployment {
	if masked := m.realToMaskedDeployment[origDeployment.GetId()]; masked != nil {
		return masked
	}

	var maskedNS string
	if maskedNS = m.realToMaskedNamespace[origDeployment.GetNamespace()]; maskedNS == "" {
		maskedNS = fmt.Sprintf("%s #%d", MaskedNamespaceName, len(m.realToMaskedNamespace)+1)
		m.realToMaskedNamespace[origDeployment.GetNamespace()] = maskedNS
	}

	masked := &storage.ListDeployment{
		Id:        uuid.NewV4().String(),
		Name:      fmt.Sprintf("%s #%d", MaskedDeploymentName, len(m.realToMaskedDeployment)+1),
		Namespace: maskedNS,
		Cluster:   origDeployment.GetCluster(),
		ClusterId: origDeployment.GetClusterId(),
	}
	m.realToMaskedDeployment[origDeployment.GetId()] = masked
	m.maskedDeployments = append(m.maskedDeployments, masked)
	return masked
}

func (m *flowGraphMasker) GetMaskedDeployments() []*storage.ListDeployment {
	return m.maskedDeployments
}
