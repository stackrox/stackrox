package controller

import (
	"github.com/np-guard/cluster-topology-analyzer/pkg/analyzer"
	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// Scans the given directory for YAMLs with k8s resources and extracts required connections between workloads
func extractConnections(dirPath string, stopOn1stErr bool) ([]common.Resource, []*common.Connections, []FileProcessingError) {
	// 1. Get all relevant resources from the repo and parse them
	dObjs, fileErrors := getK8sDeploymentResources(dirPath, stopOn1stErr)
	if stopProcessing(stopOn1stErr, fileErrors) {
		return nil, nil, fileErrors
	}
	if len(dObjs) == 0 {
		fileErrors = appendAndLogNewError(fileErrors, noK8sResourcesFound())
		return []common.Resource{}, []*common.Connections{}, fileErrors
	}

	resources, links, parseErrors := parseResources(dObjs)
	fileErrors = append(fileErrors, parseErrors...)
	if stopProcessing(stopOn1stErr, fileErrors) {
		return nil, nil, fileErrors
	}

	// 2. Discover all connections between resources
	connections := discoverConnections(resources, links)
	return resources, connections, fileErrors
}

func parseResources(objs []parsedK8sObjects) ([]common.Resource, []common.Service, []FileProcessingError) {
	resources := []common.Resource{}
	links := []common.Service{}
	configmaps := map[string]common.CfgMap{} // map from a configmap's full-name to its data
	parseErrors := []FileProcessingError{}
	for _, o := range objs {
		r, l, c, e := parseResource(o)
		resources = append(resources, r...)
		links = append(links, l...)
		parseErrors = append(parseErrors, e...)
		for _, cfgObj := range c {
			configmaps[cfgObj.FullName] = cfgObj
		}
	}
	for idx := range resources {
		res := &resources[idx]

		// handle config maps data to be associated into relevant deployments resource objects
		for _, cfgMapRef := range res.Resource.ConfigMapRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapRef
			if cfgMap, ok := configmaps[configmapFullName]; ok {
				for _, v := range cfgMap.Data {
					if analyzer.IsNetworkAddressValue(v) {
						res.Resource.Envs = append(res.Resource.Envs, v)
					}
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name))
			}
		}
		for _, cfgMapKeyRef := range res.Resource.ConfigMapKeyRefs {
			configmapFullName := res.Resource.Namespace + "/" + cfgMapKeyRef.Name
			if cfgMap, ok := configmaps[configmapFullName]; ok {
				if val, ok := cfgMap.Data[cfgMapKeyRef.Key]; ok {
					if analyzer.IsNetworkAddressValue(val) {
						res.Resource.Envs = append(res.Resource.Envs, val)
					}
				} else {
					parseErrors = appendAndLogNewError(parseErrors, configMapKeyNotFound(cfgMapKeyRef.Name, cfgMapKeyRef.Key, res.Resource.Name))
				}
			} else {
				parseErrors = appendAndLogNewError(parseErrors, configMapNotFound(configmapFullName, res.Resource.Name))
			}
		}
	}

	return resources, links, parseErrors
}

func parseResource(obj parsedK8sObjects) ([]common.Resource, []common.Service, []common.CfgMap, []FileProcessingError) {
	links := []common.Service{}
	deployments := []common.Resource{}
	configMaps := []common.CfgMap{}
	parseErrors := []FileProcessingError{}

	for _, p := range obj.DeployObjects {
		switch p.GroupKind {
		case service:
			res, err := analyzer.ScanK8sServiceObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err))
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			links = append(links, res)
		case configmap:
			res, err := analyzer.ScanK8sConfigmapObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err))
				continue
			}
			configMaps = append(configMaps, res)
		default:
			res, err := analyzer.ScanK8sWorkloadObject(p.GroupKind, p.RuntimeObject)
			if err != nil {
				parseErrors = appendAndLogNewError(parseErrors, failedScanningResource(p.GroupKind, obj.ManifestFilepath, err))
				continue
			}
			res.Resource.FilePath = obj.ManifestFilepath
			deployments = append(deployments, res)
		}
	}

	return deployments, links, configMaps, parseErrors
}
