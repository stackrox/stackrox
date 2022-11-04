package controller

import (
	"fmt"
	"strings"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// This function is at the core of the topology analysis
// For each resource, it finds other resources that may use it and compiles a list of connections holding these dependencies
func discoverConnections(resources []common.Resource, links []common.Service) []*common.Connections {
	connections := []*common.Connections{}
	for destResIdx := range resources {
		destRes := &resources[destResIdx]
		deploymentServices := findServices(destRes, links)
		for svcIdx := range deploymentServices {
			svc := &deploymentServices[svcIdx]
			srcRes := findSource(resources, svc)
			if len(srcRes) > 0 {
				for _, r := range srcRes {
					activeLogger.Debugf("source: %s target: %s link: %s", svc.Resource.Name, r.Resource.Name, svc.Resource.Name)
					connections = append(connections, &common.Connections{Source: r, Target: destRes, Link: svc})
				}
			} else {
				connections = append(connections, &common.Connections{Target: destRes, Link: svc}) // indicates a source-less service
			}
		}
	}
	return connections
}

// areSelectorsContained returns true if selectors2 is contained in selectors1
func areSelectorsContained(selectors1 map[string]string, selectors2 []string) bool {
	elementMap := make(map[string]string)
	for k, v := range selectors1 {
		s := fmt.Sprintf("%s:%s", k, v)
		elementMap[s] = ""
	}
	for _, val := range selectors2 {
		_, ok := elementMap[val]
		if !ok {
			return false
		}
	}
	return true
}

// findServices returns a list of services that may be in front of a given workload resource
func findServices(resource *common.Resource, links []common.Service) []common.Service {
	var matchedSvc []common.Service
	for linkIdx := range links {
		link := &links[linkIdx]
		if link.Resource.Namespace != resource.Resource.Namespace {
			continue
		}
		// all service selector values should be contained in the input selectors of the deployment
		res := areSelectorsContained(resource.Resource.Labels, link.Resource.Selectors)
		if res {
			matchedSvc = append(matchedSvc, *link)
		}
	}

	activeLogger.Debugf("services matched to %v: %v", resource.Resource.Name, matchedSvc)
	return matchedSvc
}

// findSource returns a list of resources that are likely trying to connect to the given service
func findSource(resources []common.Resource, service *common.Service) []*common.Resource {
	tRes := []*common.Resource{}
	for resIdx := range resources {
		resource := &resources[resIdx]
		serviceAddresses := getPossibleServiceAddresses(service, resource)
		foundSrc := *resource // We copy the resource so we can specify the ports used by the source found
		matched := false
		for _, envVal := range resource.Resource.Envs {
			match, port := envValueMatchesService(envVal, service, serviceAddresses)
			if match {
				matched = true
				if port.Port > 0 {
					foundSrc.Resource.UsedPorts = append(foundSrc.Resource.UsedPorts, port)
				}
			}
		}
		if matched {
			tRes = append(tRes, &foundSrc)
		}
	}
	return tRes
}

func getPossibleServiceAddresses(service *common.Service, resource *common.Resource) []string {
	svcAddresses := []string{}
	if service.Resource.Namespace != "" {
		serviceDotNamespace := fmt.Sprintf("%s.%s", service.Resource.Name, service.Resource.Namespace)
		svcAddresses = append(svcAddresses, serviceDotNamespace, serviceDotNamespace+".svc.cluster.local")
	}
	if service.Resource.Namespace == resource.Resource.Namespace { // both service and resource live in the same namespace
		svcAddresses = append(svcAddresses, service.Resource.Name)
	}

	return svcAddresses
}

func envValueMatchesService(envVal string, service *common.Service, serviceAddresses []string) (bool, common.SvcNetworkAttr) {
	envVal = strings.TrimPrefix(envVal, "http://")
	envVal = strings.TrimPrefix(envVal, "https://")

	// first look for matches without specified port
	for _, svcAddress := range serviceAddresses {
		if svcAddress == envVal {
			return true, common.SvcNetworkAttr{} // this means no specified port
		}
	}

	// Now look for matches that have port specified
	for _, p := range service.Resource.Network {
		for _, svcAddress := range serviceAddresses {
			serviceWithPort := fmt.Sprintf("%s:%d", svcAddress, p.Port)
			if envVal == serviceWithPort {
				return true, p
			}
		}
	}
	return false, common.SvcNetworkAttr{}
}
