package common

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// MitreAttackReadOnlyStore provides functionality to read MITRE ATT&CK vectors.
// A vector represents MITRE tactics (why) and its techniques/sub-techniques (how).
type MitreAttackReadOnlyStore interface {
	GetAll() []*storage.MitreAttackVector
	Get(id string) (*storage.MitreAttackVector, error)
}

// mitreAttackStore provides functionality to read and write MITRE ATT&CK vectors.
type mitreAttackStore interface {
	MitreAttackReadOnlyStore
	// adds the vector and overwrites if already present.
	add(id string, vector *storage.MitreAttackVector)
}

type mitreAttackStoreImpl struct {
	// mitreAttackVectors stores MITRE ATT&CK vectors keyed by tactic ID.
	mitreAttackVectors map[string]*storage.MitreAttackVector
}

func newMitreAttackStore() *mitreAttackStoreImpl {
	return &mitreAttackStoreImpl{
		mitreAttackVectors: make(map[string]*storage.MitreAttackVector),
	}
}

func (s *mitreAttackStoreImpl) GetAll() []*storage.MitreAttackVector {
	resp := make([]*storage.MitreAttackVector, 0, len(s.mitreAttackVectors))
	for _, vector := range s.mitreAttackVectors {
		resp = append(resp, vector)
	}

	sort.Slice(resp, func(i, j int) bool {
		return resp[i].GetTactic().GetName() < resp[j].GetTactic().GetName()
	})

	return resp
}

func (s *mitreAttackStoreImpl) Get(id string) (*storage.MitreAttackVector, error) {
	if id == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "MITRE ATT&CK tactic ID must be provided")
	}

	v := s.mitreAttackVectors[id]
	if v == nil {
		return nil, errors.Wrapf(errorhelpers.ErrNotFound, "MITRE ATT&CK vector for tactic %q not found. Please check the tactic ID and retry.", id)
	}
	return v, nil
}

func (s *mitreAttackStoreImpl) add(id string, vector *storage.MitreAttackVector) {
	s.mitreAttackVectors[id] = vector
}
