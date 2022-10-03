package clustertype

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/utils"
)

type wrapper struct {
	ClusterType *storage.ClusterType
}

var (
	clusterStringToType = map[string]storage.ClusterType{
		"k8s":        storage.ClusterType_KUBERNETES_CLUSTER,
		"openshift":  storage.ClusterType_OPENSHIFT_CLUSTER,
		"openshift4": storage.ClusterType_OPENSHIFT4_CLUSTER,
	}

	clusterEnumToString = utils.InvertMap(clusterStringToType)

	validClusterStrings = func() []string {
		out := make([]string, 0, len(clusterStringToType))
		for s := range clusterStringToType {
			out = append(out, s)
		}
		return out
	}()
)

func (w wrapper) String() string {
	return clusterEnumToString[*w.ClusterType]
}

func (w wrapper) Set(input string) error {
	if val, ok := clusterStringToType[strings.ToLower(input)]; ok {
		*w.ClusterType = val
		return nil
	}
	return errox.InvalidArgs.Newf("invalid cluster type: %q; valid values are %+v", input, validClusterStrings)
}

func (w wrapper) Type() string {
	return "cluster type"
}
