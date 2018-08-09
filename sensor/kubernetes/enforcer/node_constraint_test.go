package enforcer

import (
	"testing"

	"github.com/stackrox/rox/pkg/enforcers"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stretchr/testify/assert"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func TestApplyNodeConstraint(t *testing.T) {
	nodeConstraints := make(map[string]map[string]string)

	deployment := &v1beta1.Deployment{}
	applyNodeConstraintToObj(deployment, "alertID")
	nodeConstraints[pkgKubernetes.Deployment] = deployment.Spec.Template.Spec.NodeSelector

	daemonSet := &v1beta1.DaemonSet{}
	applyNodeConstraintToObj(daemonSet, "alertID")
	nodeConstraints[pkgKubernetes.DaemonSet] = daemonSet.Spec.Template.Spec.NodeSelector

	replicaSet := &v1beta1.ReplicaSet{}
	applyNodeConstraintToObj(replicaSet, "alertID")
	nodeConstraints[pkgKubernetes.ReplicaSet] = replicaSet.Spec.Template.Spec.NodeSelector

	replicationController := &v1.ReplicationController{
		Spec: v1.ReplicationControllerSpec{
			Template: &v1.PodTemplateSpec{},
		},
	}
	applyNodeConstraintToObj(replicationController, "alertID")
	nodeConstraints[pkgKubernetes.ReplicationController] = replicationController.Spec.Template.Spec.NodeSelector

	statefulSet := &appsv1beta1.StatefulSet{}
	applyNodeConstraintToObj(statefulSet, "alertID")
	nodeConstraints[pkgKubernetes.StatefulSet] = statefulSet.Spec.Template.Spec.NodeSelector

	for resourceType, constraint := range nodeConstraints {
		t.Run(resourceType, func(t *testing.T) {
			assert.NotNil(t, constraint)
			assert.Contains(t, constraint, enforcers.UnsatisfiableNodeConstraintKey)
			assert.Equal(t, constraint[enforcers.UnsatisfiableNodeConstraintKey], "alertID")
		})
	}
}
