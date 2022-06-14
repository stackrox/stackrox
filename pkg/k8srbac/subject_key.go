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
		name:      sub.Name,
		clusterID: sub.ClusterId,
		namespace: sub.Namespace,
		kind:      sub.Kind,
	}
}
