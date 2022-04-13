package k8srbac

import (
	"sort"

	"github.com/stackrox/stackrox/generated/storage"
)

// SubjectSet holds a deduplicated set of Subjects.
type SubjectSet interface {
	Add(subs ...*storage.Subject)
	Contains(sub ...*storage.Subject) bool
	ContainsSet(inputSet SubjectSet) bool
	ToSlice() []*storage.Subject
	Cardinality() int
}

// NewSubjectSet returns a new SubjectSet instance.
func NewSubjectSet(subs ...*storage.Subject) SubjectSet {
	ss := &subjectSet{
		subjectsByKey: make(map[subjectKey]*storage.Subject),
	}
	ss.Add(subs...)
	return ss
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
func (ss *subjectSet) Contains(subs ...*storage.Subject) bool {
	for _, sub := range subs {
		key := keyForSubject(sub)
		if _, contains := ss.subjectsByKey[key]; !contains {
			return false
		}
	}
	return true
}

// ContainsSet returns whether the set contains all subjects in the input set.
func (ss *subjectSet) ContainsSet(inputSet SubjectSet) bool {
	if ss.Cardinality() < inputSet.Cardinality() {
		return false
	}
	return ss.Contains(inputSet.ToSlice()...)
}

// Cardinality returns the current count of subjects in the set.
func (ss *subjectSet) Cardinality() int {
	return len(ss.subjectsByKey)
}

func getSortedSubjectList(subjectsByKey map[subjectKey]*storage.Subject) []*storage.Subject {
	// Sort for stability
	sortedSubjects := make([]*storage.Subject, 0, len(subjectsByKey))
	for _, subj := range subjectsByKey {
		sortedSubjects = append(sortedSubjects, subj)
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
