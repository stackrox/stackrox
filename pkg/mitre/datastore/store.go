package datastore

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mitre"
)

var (
	log = logging.LoggerForModule()
)

// AttackReadOnlyDataStore provides functionality to read MITRE ATT&CK vectors.
// A vector represents MITRE tactics (why) and its techniques/sub-techniques (how).
//
//go:generate mockgen-wrapper
type AttackReadOnlyDataStore interface {
	GetAll() []*storage.MitreAttackVector
	Get(id string) (*storage.MitreAttackVector, error)
}

type mitreAttackStoreImpl struct {
	// mitreAttackVectors stores MITRE ATT&CK vectors keyed by tactic ID.
	mitreAttackVectors map[string]*storage.MitreAttackVector
}

// NewMitreAttackStore reads the json into the mitre attack vectors in memory map
func NewMitreAttackStore() AttackReadOnlyDataStore {
	s := &mitreAttackStoreImpl{
		mitreAttackVectors: make(map[string]*storage.MitreAttackVector),
	}
	// If ATT&CK data cannot be loaded, fail open.
	if err := s.loadBundledData(); err != nil {
		log.Errorf("MITRE ATT&CK data for system policies unavailable: %v", err)
	}
	return s
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
		return nil, errors.Wrap(errox.InvalidArgs, "MITRE ATT&CK tactic ID must be provided")
	}

	v := s.mitreAttackVectors[id]
	if v == nil {
		return nil, errors.Wrapf(errox.NotFound, "MITRE ATT&CK vector for tactic %q not found. Please check the tactic ID and retry.", id)
	}
	return v, nil
}

func (s *mitreAttackStoreImpl) loadBundledData() error {
	attackBundle, err := mitre.GetMitreBundle()
	if err != nil {
		return errors.Wrap(err, "loading default MITRE ATT&CK data")
	}

	// Flatten vectors from all matrices since we populate all enterprise.
	vectors := mitre.FlattenMitreMatrices(attackBundle.GetMatrices()...)
	for _, vector := range vectors {
		s.mitreAttackVectors[vector.GetTactic().GetId()] = vector
	}
	return nil
}
