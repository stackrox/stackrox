package listener

import (
	pkgV1 "bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/golang/protobuf/ptypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type resourceWrap struct {
	metav1.ObjectMeta
	resourceType string

	replicas int
	images   []string
	action   pkgV1.ResourceAction
}

func (r *resourceWrap) asDeploymentEvent() *pkgV1.DeploymentEvent {
	updatedTime, err := ptypes.TimestampProto(r.CreationTimestamp.Time)
	if err != nil {
		logger.Error(err)
	}

	imgSlice := make([]*pkgV1.Image, len(r.images))
	for i, img := range r.images {
		imgSlice[i] = images.GenerateImageFromString(img)
	}

	return &pkgV1.DeploymentEvent{
		Deployment: &pkgV1.Deployment{
			Id:        string(r.UID),
			Name:      r.Name,
			Version:   r.ResourceVersion,
			Type:      r.resourceType,
			Replicas:  int64(r.replicas),
			UpdatedAt: updatedTime,
			Images:    imgSlice,
		},
		Action: r.action,
	}
}
