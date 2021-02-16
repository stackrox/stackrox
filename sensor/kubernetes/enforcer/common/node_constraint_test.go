package common

import (
	"testing"

	"github.com/stackrox/rox/pkg/detection/deploytime"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func TestApplyNodeConstraint(t *testing.T) {
	a := assert.New(t)
	nodeConstraints := make(map[string]map[string]string)

	deployment := &appsV1.Deployment{}
	a.NoError(ApplyNodeConstraintToObj(deployment, "alertID"))
	nodeConstraints[pkgKubernetes.Deployment] = deployment.Spec.Template.Spec.NodeSelector

	daemonSet := &appsV1.DaemonSet{}
	a.NoError(ApplyNodeConstraintToObj(daemonSet, "alertID"))
	nodeConstraints[pkgKubernetes.DaemonSet] = daemonSet.Spec.Template.Spec.NodeSelector

	replicaSet := &appsV1.ReplicaSet{}
	a.NoError(ApplyNodeConstraintToObj(replicaSet, "alertID"))
	nodeConstraints[pkgKubernetes.ReplicaSet] = replicaSet.Spec.Template.Spec.NodeSelector

	replicationController := &v1.ReplicationController{
		Spec: v1.ReplicationControllerSpec{
			Template: &v1.PodTemplateSpec{},
		},
	}
	a.NoError(ApplyNodeConstraintToObj(replicationController, "alertID"))
	nodeConstraints[pkgKubernetes.ReplicationController] = replicationController.Spec.Template.Spec.NodeSelector

	statefulSet := &appsV1.StatefulSet{}
	a.NoError(ApplyNodeConstraintToObj(statefulSet, "alertID"))
	nodeConstraints[pkgKubernetes.StatefulSet] = statefulSet.Spec.Template.Spec.NodeSelector

	for resourceType, constraint := range nodeConstraints {
		t.Run(resourceType, func(t *testing.T) {
			assert.NotNil(t, constraint)
			assert.Contains(t, constraint, deploytime.UnsatisfiableNodeConstraintKey)
			assert.Equal(t, constraint[deploytime.UnsatisfiableNodeConstraintKey], "alertID")
		})
	}
}
