package analyzer

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/np-guard/cluster-topology-analyzer/pkg/common"
)

// Create a common.Resource object from a k8s Workload object
func ScanK8sWorkloadObject(kind string, objDataBuf []byte) (common.Resource, error) {
	var podSpecV1 v1.PodTemplateSpec
	var resourceCtx common.Resource
	var metaObj metaV1.Object
	resourceCtx.Resource.Kind = kind
	switch kind { // TODO: handle Pod
	case "ReplicaSet":
		obj := parseReplicaSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.GetLabels()
		resourceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(obj.Spec.Selector.MatchLabels)
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "ReplicationController":
		obj := parseReplicationController(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(obj.Spec.Selector)
		podSpecV1 = *obj.Spec.Template
		metaObj = obj
	case "Deployment":
		obj := parseDeployment(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(obj.Spec.Selector.MatchLabels)
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "DaemonSet":
		obj := parseDaemonSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(obj.Spec.Selector.MatchLabels)
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "StatefulSet":
		obj := parseStatefulSet(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(obj.Spec.Selector.MatchLabels)
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	case "Job":
		obj := parseJob(bytes.NewReader(objDataBuf))
		resourceCtx.Resource.Labels = obj.Spec.Template.Labels
		resourceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(obj.Spec.Selector.MatchLabels)
		podSpecV1 = obj.Spec.Template
		metaObj = obj
	default:
		return resourceCtx, fmt.Errorf("unsupported object type: `%s`", kind)
	}

	parseDeployResource(&podSpecV1, metaObj, &resourceCtx)
	return resourceCtx, nil
}

func matchLabelSelectorToStrLabels(labels map[string]string) []string {
	res := []string{}
	for k, v := range labels {
		res = append(res, fmt.Sprintf("%s:%s", k, v))
	}
	return res
}

func ScanK8sConfigmapObject(kind string, objDataBuf []byte) (common.CfgMap, error) {
	obj := parseConfigMap(bytes.NewReader(objDataBuf))
	if obj == nil {
		return common.CfgMap{}, fmt.Errorf("unable to parse configmap")
	}

	fullName := obj.ObjectMeta.Namespace + "/" + obj.ObjectMeta.Name
	data := map[string]string{}
	for k, v := range obj.Data {
		isPotentialAddress := IsNetworkAddressValue(v)
		if isPotentialAddress {
			data[k] = v
		}
	}
	return common.CfgMap{FullName: fullName, Data: data}, nil
}

// Create a common.Service object from a k8s Service object
func ScanK8sServiceObject(kind string, objDataBuf []byte) (common.Service, error) {
	if kind != "Service" {
		return common.Service{}, fmt.Errorf("expected parsing a Service resource, but got `%s`", kind)
	}

	var serviceCtx common.Service
	svcObj := parseService(bytes.NewReader(objDataBuf))
	if svcObj == nil {
		return common.Service{}, fmt.Errorf("failed to parse Service resource")
	}
	serviceCtx.Resource.Name = svcObj.GetName()
	serviceCtx.Resource.Namespace = svcObj.Namespace
	serviceCtx.Resource.Kind = kind
	serviceCtx.Resource.Type = string(svcObj.Spec.Type)
	serviceCtx.Resource.Selectors = matchLabelSelectorToStrLabels(svcObj.Spec.Selector)

	for _, p := range svcObj.Spec.Ports {
		n := common.SvcNetworkAttr{}
		n.Port = int(p.Port)
		n.TargetPort = p.TargetPort
		n.Protocol = string(p.Protocol)
		serviceCtx.Resource.Network = append(serviceCtx.Resource.Network, n)
	}

	return serviceCtx, nil
}

func parseDeployResource(podSpec *v1.PodTemplateSpec, obj metaV1.Object, resourceCtx *common.Resource) {
	resourceCtx.Resource.Name = obj.GetName()
	resourceCtx.Resource.Namespace = obj.GetNamespace()
	resourceCtx.Resource.ServiceAccountName = podSpec.Spec.ServiceAccountName
	for containerIdx := range podSpec.Spec.Containers {
		container := &podSpec.Spec.Containers[containerIdx]
		resourceCtx.Resource.Image.ID = container.Image
		for _, p := range container.Ports {
			n := common.NetworkAttr{}
			n.ContainerPort = int(p.ContainerPort)
			n.HostPort = int(p.HostPort)
			n.Protocol = string(p.Protocol)
			resourceCtx.Resource.Network = append(resourceCtx.Resource.Network, n)
		}
		for _, e := range container.Env {
			if e.Value != "" {
				if IsNetworkAddressValue(e.Value) {
					resourceCtx.Resource.Envs = append(resourceCtx.Resource.Envs, e.Value)
				}
			} else if e.ValueFrom != nil && e.ValueFrom.ConfigMapKeyRef != nil {
				keyRef := e.ValueFrom.ConfigMapKeyRef
				if keyRef.Name != "" && keyRef.Key != "" { // just store ref for now - check later if it's a network address
					cfgMapKeyRef := common.CfgMapKeyRef{Name: keyRef.Name, Key: keyRef.Key}
					resourceCtx.Resource.ConfigMapKeyRefs = append(resourceCtx.Resource.ConfigMapKeyRefs, cfgMapKeyRef)
				}
			}
		}
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil { // just store ref for now - check later if the config map values contain a network address
				resourceCtx.Resource.ConfigMapRefs = append(resourceCtx.Resource.ConfigMapRefs, envFrom.ConfigMapRef.Name)
			}
		}
	}
}

// IsNetworkAddressValue checks if a given string is a potential network address
func IsNetworkAddressValue(value string) bool {
	_, err := url.Parse(value)
	if err != nil {
		return false
	}
	_, err = strconv.Atoi(value)
	return err != nil // we do not accept integers as network addresses
}
