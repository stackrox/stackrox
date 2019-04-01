package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

type subjectKey struct {
	name      string
	namespace string
	kind      storage.SubjectKind
}

func keyForSubject(sub *storage.Subject) subjectKey {
	return subjectKey{
		name:      sub.Name,
		namespace: sub.Namespace,
		kind:      sub.Kind,
	}
}
