package service

import (
	"fmt"
	"slices"

	deploymentUtils "github.com/stackrox/rox/central/deployment/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// MaskedDeploymentName is name of a masked deployment
	MaskedDeploymentName = "masked deployment"
	// MaskedNamespaceName is name of a masked namespace
	MaskedNamespaceName = "masked namespace"
)

type flowGraphMasker struct {
	deploymentsToMask      map[string]*storage.ListDeployment
	namespaceNamesToMask   set.Set[string]
	maskedDeployments      []*storage.ListDeployment
	realToMaskedDeployment map[string]*storage.ListDeployment
	realToMaskedNamespace  map[string]string
}

func newFlowGraphMasker() *flowGraphMasker {
	return &flowGraphMasker{
		deploymentsToMask:      make(map[string]*storage.ListDeployment),
		realToMaskedDeployment: make(map[string]*storage.ListDeployment),
		realToMaskedNamespace:  make(map[string]string),
	}
}

func (m *flowGraphMasker) RegisterDeploymentForMasking(deployment *storage.ListDeployment) {
	ld := &storage.ListDeployment{}
	ld.SetId(deployment.GetId())
	ld.SetName(deployment.GetName())
	ld.SetCluster(deployment.GetCluster())
	ld.SetClusterId(deployment.GetClusterId())
	ld.SetNamespace(deployment.GetNamespace())
	m.deploymentsToMask[deployment.GetId()] = ld
	m.namespaceNamesToMask.Add(deployment.GetNamespace())
}

func (m *flowGraphMasker) MaskDeploymentsAndNamespaces() {
	orderedNamespaceNamesToMask := m.namespaceNamesToMask.AsSlice()
	slices.Sort(orderedNamespaceNamesToMask)
	for ix, ns := range orderedNamespaceNamesToMask {
		maskedNS := fmt.Sprintf("%s #%d", MaskedNamespaceName, ix+1)
		m.realToMaskedNamespace[ns] = maskedNS
	}
	m.namespaceNamesToMask.Clear()
	orderedDeploymentIDsToMask := make([]string, 0, len(m.deploymentsToMask))
	for deploymentID := range m.deploymentsToMask {
		orderedDeploymentIDsToMask = append(orderedDeploymentIDsToMask, deploymentID)
	}
	slices.Sort(orderedDeploymentIDsToMask)
	for ix, deploymentID := range orderedDeploymentIDsToMask {
		origDeployment := m.deploymentsToMask[deploymentID]
		maskedDeploymentName := fmt.Sprintf("%s #%d", MaskedDeploymentName, ix+1)
		maskedDeployment := &storage.ListDeployment{}
		maskedDeployment.SetId(deploymentUtils.GetMaskedDeploymentID(origDeployment.GetId(), origDeployment.GetName()))
		maskedDeployment.SetName(maskedDeploymentName)
		maskedDeployment.SetCluster(origDeployment.GetCluster())
		maskedDeployment.SetClusterId(origDeployment.GetClusterId())
		maskedDeployment.SetNamespace(m.realToMaskedNamespace[origDeployment.GetNamespace()])
		m.realToMaskedDeployment[deploymentID] = maskedDeployment
		m.maskedDeployments = append(m.maskedDeployments, maskedDeployment)
		delete(m.deploymentsToMask, deploymentID)
	}
}

func (m *flowGraphMasker) GetMaskedDeployment(origDeployment *storage.ListDeployment) *storage.ListDeployment {
	return m.realToMaskedDeployment[origDeployment.GetId()]
}

func (m *flowGraphMasker) GetMaskedDeployments() []*storage.ListDeployment {
	return m.maskedDeployments
}
