package platformcve

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
