package fetcher

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/cve/converter"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	pkgScanners "github.com/stackrox/rox/pkg/scanners"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Cluster),
		))
)

type orchestratorCVEManager struct {
	embeddedCVEIdToClusters map[string][]*storage.Cluster

	clusterDataStore clusterDataStore.DataStore
	cveDataStore     cveDataStore.DataStore
	cveMatcher       *cveMatcher.CVEMatcher

	creators map[string]pkgScanners.OrchestratorScannerCreator
	scanners map[string]types.OrchestratorScanner

	mutex sync.Mutex
}

func (m *orchestratorCVEManager) initialize() {
	err := m.ReconcileK8sCVEs()
	if err != nil {
		log.Errorf("failed to reconcile orchestrator CVEs: %v", err)
	}
}

func (m *orchestratorCVEManager) updateCVEs(embeddedCVEs []*storage.EmbeddedVulnerability, embeddedCVEToClusters map[string][]*storage.Cluster) error {
	newCVEIDs := set.NewStringSet()
	var newCVEs []converter.ClusterCVEParts
	for _, embeddedCVE := range embeddedCVEs {
		cve := converter.EmbeddedCVEToProtoCVE("", embeddedCVE)
		newCVEIDs.Add(cve.GetId())
		newCVEs = append(newCVEs, converter.NewClusterCVEParts(cve, embeddedCVEToClusters[embeddedCVE.GetCve()], embeddedCVE.GetFixedBy()))
	}

	m.embeddedCVEIdToClusters = embeddedCVEToClusters
	return m.updateCVEsInDB(newCVEIDs, newCVEs)
}

func (m *orchestratorCVEManager) updateCVEsInDB(cveIds set.StringSet, cves []converter.ClusterCVEParts) error {
	if err := m.cveDataStore.UpsertClusterCVEs(cveElevatedCtx, cves...); err != nil {
		return err
	}
	return reconcileCVEsInDB(m.cveDataStore, storage.CVE_K8S_CVE, cveIds)
}

// CreateOrchestratorScanner creates a types.OrchestratorScanner out of the given storage.OrchestratorIntegration.
func (m *orchestratorCVEManager) createOrchestratorScanner(source *storage.OrchestratorIntegration) (types.OrchestratorScanner, error) {
	creator, exists := m.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("scanner with type %q does not exist", source.GetType())
	}
	scanner, err := creator(source)
	if err != nil {
		return nil, err
	}
	return scanner, nil
}

func (m *orchestratorCVEManager) UpsertOrchestratorScanner(integration *storage.OrchestratorIntegration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	scanner, err := m.createOrchestratorScanner(integration)
	if err != nil {
		return errors.Wrap(err, "Failed to create orchestrator scanner")
	}
	m.scanners[integration.GetId()] = scanner
	return nil
}

func (m *orchestratorCVEManager) RemoveIntegration(integrationID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.scanners, integrationID)
}

// Scan calls orchestrator scanner to scan the CVEs for kube vulnerabilities.
func (m *orchestratorCVEManager) Scan(version string) ([]*storage.EmbeddedVulnerability, error) {
	errorList := errorhelpers.NewErrorList(fmt.Sprintf("error scanning orchestrator for Kubernetes:%s", version))
	if len(m.scanners) == 0 {
		errorList.AddError(errors.New("no orchestrator scanners are integrated"))
		return nil, errorList.ToError()
	}

	var allVulns []*storage.EmbeddedVulnerability
	for _, scanner := range m.scanners {
		result, err := scanner.KubernetesScan(version)
		if err != nil {
			errorList.AddError(err)
			continue
		}
		vulnIDsSet := set.NewStringSet()
		for _, v := range result {
			for _, vuln := range v {
				if vulnIDsSet.Add(vuln.GetCve()) {
					allVulns = append(allVulns, vuln)
				}
			}
		}
		return allVulns, nil
	}

	return nil, errorList.ToError()
}

// ReconcileK8sCVEs fetches new CVEs from scanner and reconciles them
func (m *orchestratorCVEManager) ReconcileK8sCVEs() error {
	clusters, err := m.clusterDataStore.GetClusters(readCtx)
	if err != nil {
		return err
	}

	log.Infof("Found %d clusters to scan for orchestrator vulnerabilities.", len(clusters))
	versionToClusters := make(map[string][]*storage.Cluster)
	for _, cluster := range clusters {
		version := cluster.GetStatus().GetOrchestratorMetadata().GetVersion()
		versionToClusters[version] = append(versionToClusters[version], cluster)
	}

	embeddedCVEIDToClusters := make(map[string][]*storage.Cluster)
	var allEmbeddedCVEs []*storage.EmbeddedVulnerability
	for version := range versionToClusters {
		vulns, err := m.Scan(version)
		if err != nil {
			return err
		}
		for _, vuln := range vulns {
			if _, ok := embeddedCVEIDToClusters[vuln.GetCve()]; !ok {
				allEmbeddedCVEs = append(allEmbeddedCVEs, vuln)
			}
			embeddedCVEIDToClusters[vuln.GetCve()] = append(embeddedCVEIDToClusters[vuln.GetCve()], versionToClusters[version]...)
		}
	}

	err = m.updateCVEs(allEmbeddedCVEs, embeddedCVEIDToClusters)
	if err != nil {
		return err
	}
	log.Infof("Successfully fetched %d k8s CVEs", len(m.embeddedCVEIdToClusters))
	return nil
}

func (m *orchestratorCVEManager) getAffectedClusters(cveID string) ([]*storage.Cluster, error) {
	if clusters, ok := m.embeddedCVEIdToClusters[cveID]; ok {
		return clusters, nil
	}
	return nil, errors.Errorf("Cannot find cve with id %s", cveID)
}
