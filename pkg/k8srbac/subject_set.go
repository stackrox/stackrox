package k8srbac

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
)

// SubjectSet holds a deduplicating set of Subjects.
type SubjectSet interface {
	Add(subs ...*storage.Subject)
	Contains(sub *storage.Subject) bool
	ToSlice() []*storage.Subject
}

// NewSubjectSet returns a new SubjectSet instance.
func NewSubjectSet() SubjectSet {
	return &subjectSet{
		subjectsByKey: make(map[subjectKey]*storage.Subject),
	}
}

type subjectSet struct {
	subjectsByKey map[subjectKey]*storage.Subject
}

// AddAll adds all of the inputs to the set.
func (ss *subjectSet) Add(subs ...*storage.Subject) {
	for _, sub := range subs {
		ss.subjectsByKey[keyForSubject(sub)] = sub
	}
}

// ToSlice returns the list of Subjects sorted by type, namespace, and name.
func (ss *subjectSet) ToSlice() []*storage.Subject {
	if len(ss.subjectsByKey) == 0 {
		return nil
	}
	return getSortedSubjectList(ss.subjectsByKey)
}

// Contains returns if the set contains the given subject.
func (ss *subjectSet) Contains(sub *storage.Subject) bool {
	key := keyForSubject(sub)
	_, contains := ss.subjectsByKey[key]
	return contains
}

func getSortedSubjectList(subjectsByKey map[subjectKey]*storage.Subject) []*storage.Subject {
	// Sort for stability
	sortedSubjects := make([]*storage.Subject, 0, len(subjectsByKey))
	for _, subject := range subjectsByKey {
		sortedSubjects = append(sortedSubjects, subject)
	}
	sort.SliceStable(sortedSubjects, func(idx1, idx2 int) bool {
		return subjectIsLess(sortedSubjects[idx1], sortedSubjects[idx2])
	})
	return sortedSubjects
}

func subjectIsLess(sub1, sub2 *storage.Subject) bool {
	if sub1.Kind != sub2.Kind {
		return sub1.Kind < sub2.Kind
	}
	if sub1.Namespace != sub2.Namespace {
		return sub1.Namespace < sub2.Namespace
	}
	return sub1.Name < sub2.Name
}
