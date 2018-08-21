package dnrintegration

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

const (
	dnrRoxClusterIDEnv = "ROX_CLUSTER_ID"
)

func (d *dnrIntegrationImpl) initialize(requiredPreventClusterIDs []string) error {
	if len(requiredPreventClusterIDs) == 0 {
		return errors.New("empty cluster id list received")
	}

	version, err := d.version()
	if err != nil {
		return err
	}

	// It's safe to assume nobody is using a version older than this.
	// Checking against old releases explicitly will make sure that PR tags + latest etc default to
	// being treated as multi-cluster.
	if strings.HasPrefix(version, "2.0") || strings.HasPrefix(version, "2.1") {
		if len(requiredPreventClusterIDs) > 1 {
			return fmt.Errorf("D&R version is %s, which does not support multi-cluster", version)
		}
		d.isPreMultiCluster = true
		return nil
	}

	d.clusterIDMapping = make(map[string]string)

	// To get the D&R cluster id corresponding to a Prevent cluster, we look for stackrox/collector deployments
	// in that cluster, and inspect its environment for the ROX_CLUSTER_ID value.
	for _, preventClusterID := range requiredPreventClusterIDs {
		deployments, err := d.deploymentStore.SearchRawDeployments(&v1.ParsedSearchRequest{Fields: map[string]*v1.ParsedSearchRequest_Values{
			search.ClusterID:      {Values: []string{preventClusterID}},
			search.DeploymentName: {Values: []string{"collector"}},
			search.Namespace:      {Values: []string{"stackrox"}},
		}})
		if err != nil {
			return fmt.Errorf("couldn't search for collector deployments in cluster '%s': %s", preventClusterID, err)
		}
		if len(deployments) == 0 {
			return fmt.Errorf("couldn't find deployment stackrox/collector in cluster '%s'; D&R possibly not deployed there",
				preventClusterID)
		}
		// This should basically never happen.
		if len(deployments) > 1 {
			deploymentNames := make([]string, 0, len(deployments))
			for _, deployment := range deployments {
				deploymentNames = append(deploymentNames, deployment.GetName())
			}
			return fmt.Errorf("found multiple deployments matching query stackrox/collector in cluster '%s': %#v", preventClusterID,
				deploymentNames)
		}

		collectorContainers := deployments[0].GetContainers()

		// The collector deployment spec always has only one container, so this should never happen either.
		if len(collectorContainers) != 1 {
			return fmt.Errorf("found %d containers in collector deployment for cluster '%s', expected 1",
				len(collectorContainers), preventClusterID)
		}

		var dnrClusterID string
		for _, keyValuePair := range collectorContainers[0].GetConfig().GetEnv() {
			if keyValuePair.GetKey() == dnrRoxClusterIDEnv {
				dnrClusterID = keyValuePair.GetValue()
				break
			}
		}

		if dnrClusterID == "" {
			return fmt.Errorf("couldn't find D&R cluster corresponding to Prevent cluster '%s'", preventClusterID)
		}
		d.clusterIDMapping[preventClusterID] = dnrClusterID
	}

	return nil
}

func (d *dnrIntegrationImpl) addServiceToMapping(m serviceMapping, svc service, dnrClusterID string) {
	// D&R sometimes has the namespace unset if it is default, while Prevent always sets it to "default". (example: on Swarm)
	namespace := svc.Namespace
	if namespace == "" {
		namespace = "default"
	}
	m[preventDeploymentMetadata{name: svc.Name, namespace: namespace}] = dnrServiceMetadata{serviceID: svc.ID, clusterID: dnrClusterID}
}

func (d *dnrIntegrationImpl) refreshServiceMappingSingleCluster() error {
	services, err := d.Services(url.Values{})
	if err != nil {
		return err
	}

	mapping := make(serviceMapping, len(services))
	for _, service := range services {
		d.addServiceToMapping(mapping, service, "")
	}

	d.serviceMappingSingleCluster = mapping
	return nil
}

func (d *dnrIntegrationImpl) refreshServiceMappingForCluster(preventClusterID, dnrClusterID string) error {
	params := url.Values{}
	params.Set("cluster_id", dnrClusterID)
	services, err := d.Services(params)
	if err != nil {
		return err
	}

	mapping := make(serviceMapping, len(services))
	for _, service := range services {
		d.addServiceToMapping(mapping, service, dnrClusterID)
	}

	d.serviceMappingsLock.Lock()
	defer d.serviceMappingsLock.Unlock()
	d.serviceMappings[preventClusterID] = mapping
	return nil
}

func (d *dnrIntegrationImpl) refreshServiceMappings() error {
	if d.isPreMultiCluster {
		return d.refreshServiceMappingSingleCluster()
	}

	d.serviceMappingsLock.Lock()
	if d.serviceMappings == nil {
		d.serviceMappings = make(map[string]serviceMapping)
	}
	d.serviceMappingsLock.Unlock()

	for preventClusterID, dnrClusterID := range d.clusterIDMapping {
		err := d.refreshServiceMappingForCluster(preventClusterID, dnrClusterID)
		if err != nil {
			return err
		}
	}
	return nil
}

// Rate limit the service mapping so that we don't bring down D&R.
func (d *dnrIntegrationImpl) refreshServiceMappingRateLimited() {
	if d.serviceMappingsRateLimiter.Allow() {
		err := d.refreshServiceMappings()
		if err != nil {
			logger.Errorf("Failed to refresh Prevent<->D&R service mapping: %s", err)
		}
	}
}

func (d *dnrIntegrationImpl) serviceMappingForCluster(clusterID string) (mapping serviceMapping, found bool) {
	d.refreshServiceMappingRateLimited()
	if d.isPreMultiCluster {
		if d.serviceMappingSingleCluster == nil {
			return nil, false
		}
		return d.serviceMappingSingleCluster, true
	}

	d.serviceMappingsLock.RLock()
	defer d.serviceMappingsLock.RUnlock()
	mapping, found = d.serviceMappings[clusterID]
	return
}

func (d *dnrIntegrationImpl) getDNRServiceParams(clusterID, namespace, serviceName string) (params url.Values, found bool) {
	mapping, found := d.serviceMappingForCluster(clusterID)
	if !found {
		return
	}

	dnr := mapping[preventDeploymentMetadata{name: serviceName, namespace: namespace}]

	params = url.Values{}
	params.Set("serviceID", dnr.serviceID)
	if dnr.clusterID != "" {
		params.Set("cluster_id", dnr.clusterID)
	}
	return
}
