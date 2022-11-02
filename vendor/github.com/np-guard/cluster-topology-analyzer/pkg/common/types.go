package common

import (
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CfgMap struct {
	FullName string
	Data     map[string]string
}

type CfgMapKeyRef struct {
	Name string
	Key  string
}

type Resource struct {
	Resource struct {
		Name               string            `json:"name,omitempty"`
		Namespace          string            `json:"namespace,omitempty"`
		Selectors          []string          `json:"selectors,omitempty"`
		Labels             map[string]string `json:"labels,omitempty"`
		ServiceAccountName string            `json:"serviceaccountname,omitempty"`
		FilePath           string            `json:"filepath,omitempty"`
		Kind               string            `json:"kind,omitempty"`
		Image              struct {
			ID string `json:"id,omitempty"`
		} `json:"image"`
		Network          []NetworkAttr `json:"network"`
		Envs             []string
		ConfigMapRefs    []string       `json:"-"`
		ConfigMapKeyRefs []CfgMapKeyRef `json:"-"`
		UsedPorts        []SvcNetworkAttr
	} `json:"resource,omitempty"`
}

type NetworkAttr struct {
	HostPort      int    `json:"host_port,omitempty"`
	ContainerPort int    `json:"container_url,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
}

type SvcNetworkAttr struct {
	Port       int                `json:"port,omitempty"`
	TargetPort intstr.IntOrString `json:"target_port,omitempty"`
	Protocol   string             `json:"protocol,omitempty"`
}

type Service struct {
	Resource struct {
		Name      string   `json:"name,omitempty"`
		Namespace string   `json:"namespace,omitempty"`
		Selectors []string `json:"selectors,omitempty"`
		// Labels    map[string]string `json:"labels, omitempty"`
		Type     string           `json:"type,omitempty"`
		FilePath string           `json:"filepath,omitempty"`
		Kind     string           `json:"kind,omitempty"`
		Network  []SvcNetworkAttr `json:"network,omitempty"`
	} `json:"resource,omitempty"`
}

type Connections struct {
	Source *Resource `json:"source,omitempty"`
	Target *Resource `json:"target"`
	Link   *Service  `json:"link"`
}

const (
	ServiceCtx = "service"
	DeployCtx  = "deployment"
)
