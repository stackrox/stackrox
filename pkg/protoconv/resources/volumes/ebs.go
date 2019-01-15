package volumes

import "k8s.io/api/core/v1"

const ebsType = "AWSElasticBlockStore"

type ebs struct {
	*v1.AWSElasticBlockStoreVolumeSource
}

func (e *ebs) Source() string {
	return e.VolumeID
}

func (e *ebs) Type() string {
	return ebsType
}

func createEBS(i interface{}) VolumeSource {
	ebsVolume, ok := i.(*v1.AWSElasticBlockStoreVolumeSource)
	if !ok {
		return &Unimplemented{}
	}
	return &ebs{
		AWSElasticBlockStoreVolumeSource: ebsVolume,
	}
}

func init() {
	VolumeRegistry[ebsType] = createEBS
}
