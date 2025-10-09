package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

type subjectKey struct {
	name      string
	clusterID string
	namespace string
	kind      storage.SubjectKind
}

func keyForSubject(sub *storage.Subject) subjectKey {
	return subjectKey{
		name:      sub.GetName(),
		clusterID: sub.GetClusterId(),
		namespace: sub.GetNamespace(),
		kind:      sub.GetKind(),
	}
}
