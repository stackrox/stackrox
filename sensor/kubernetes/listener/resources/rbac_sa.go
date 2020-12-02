package resources

import (
	"errors"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/utils"
)

func (rs *rbacUpdaterImpl) assignPermissionLevelToDeployment(wrap *deploymentWrap) {
	subject := &storage.Subject{
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
		Name:      wrap.GetServiceAccount(),
		Namespace: wrap.GetNamespace(),
	}

	rs.lock.Lock()
	defer rs.lock.Unlock()
	if !rs.hasBuiltInitialBucket {
		rs.hasBuiltInitialBucket = rs.rebuildEvaluatorBucketsNoLock()
		if !rs.hasBuiltInitialBucket {
			utils.Should(errors.New("deployment permissions should not be evaluated if rbac has not been synced"))
		}
	}
	wrap.ServiceAccountPermissionLevel = rs.bucketEvaluator.getBucketNoLock(subject)
}

// Evaluate the permission bucket for a subject.
////////////////////////////////////////////////
type bucketEvaluator struct {
	clusterEvaluator    k8srbac.Evaluator
	namespaceEvaluators map[string]k8srbac.Evaluator
}

func newBucketEvaluator(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) *bucketEvaluator {
	return &bucketEvaluator{
		clusterEvaluator:    k8srbac.MakeClusterEvaluator(roles, bindings),
		namespaceEvaluators: k8srbac.MakeNamespaceEvaluators(roles, bindings),
	}
}

func (be *bucketEvaluator) getBucketNoLock(subject *storage.Subject) storage.PermissionLevel {
	// Check for admin or elevated permissions cluster wide.
	clusterPermissions := be.clusterEvaluator.ForSubject(subject)
	if clusterPermissions.Grants(k8srbac.EffectiveAdmin) {
		return storage.PermissionLevel_CLUSTER_ADMIN
	}
	if k8srbac.CanWriteAResource(clusterPermissions) || k8srbac.CanReadAResource(clusterPermissions) {
		return storage.PermissionLevel_ELEVATED_CLUSTER_WIDE
	}

	// Check for elevated or default permissions within a namespace.
	maxPermissions := storage.PermissionLevel_NONE
	for _, namespaceEvaluator := range be.namespaceEvaluators {
		if namespaceEvaluator == nil {
			continue
		}
		namespacePermissions := namespaceEvaluator.ForSubject(subject)
		if k8srbac.CanWriteAResource(namespacePermissions) || namespacePermissions.Grants(k8srbac.ListAnything) {
			return storage.PermissionLevel_ELEVATED_IN_NAMESPACE
		} else if k8srbac.CanReadAResource(namespacePermissions) {
			maxPermissions = storage.PermissionLevel_DEFAULT
		}
	}
	return maxPermissions
}
