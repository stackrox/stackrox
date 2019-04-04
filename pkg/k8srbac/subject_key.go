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

func subjectForKey(key subjectKey) *storage.Subject {
	return &storage.Subject{
		Name:      key.name,
		Namespace: key.namespace,
		Kind:      key.kind,
	}
}
