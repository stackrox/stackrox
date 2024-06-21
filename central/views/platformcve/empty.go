package platformcve

import "github.com/stackrox/rox/central/views/common"

// NewEmptyClusterCountByPlatformType creates an empty instance of type ClusterCountByPlatformType.
func NewEmptyClusterCountByPlatformType() ClusterCountByPlatformType {
	return &emptyClusterCountByPlatformType{}
}

type emptyClusterCountByPlatformType struct{}

func (e *emptyClusterCountByPlatformType) GetGenericClusterCount() int {
	return 0
}

func (e *emptyClusterCountByPlatformType) GetKubernetesClusterCount() int {
	return 0
}

func (e *emptyClusterCountByPlatformType) GetOpenshiftClusterCount() int {
	return 0
}

func (e *emptyClusterCountByPlatformType) GetOpenshift4ClusterCount() int {
	return 0
}

func NewEmptyCVECountByType() CVECountByType {
	return &emptyCVECountByType{}
}

type emptyCVECountByType struct{}

func (e *emptyCVECountByType) GetKubernetesCVECount() int {
	return 0
}

func (e *emptyCVECountByType) GetOpenshiftCVECount() int {
	return 0
}

func (e *emptyCVECountByType) GetIstioCVECount() int {
	return 0
}

func NewEmptyCVECountByFixability() common.ResourceCountByFixability {
	return &emptyCVECountByFixability{}
}

type emptyCVECountByFixability struct{}

func (e *emptyCVECountByFixability) GetTotal() int {
	return 0
}

func (e *emptyCVECountByFixability) GetFixable() int {
	return 0
}
