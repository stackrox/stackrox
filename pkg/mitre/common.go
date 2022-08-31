package mitre

import (
	"embed"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

const (
	mitreBundleFile = "files/mitre.json"
)

var (
	//go:embed files/mitre.json
	mitreFS embed.FS
)

// GetMitreBundle returns MITRE ATT&CK bundle.
func GetMitreBundle() (*storage.MitreAttackBundle, error) {
	bytes, err := mitreFS.ReadFile(mitreBundleFile)
	if err != nil {
		return nil, errors.Wrapf(err, "could not load MITRE ATT&CK data from %q", mitreBundleFile)
	}

	var bundle storage.MitreAttackBundle
	if err := json.Unmarshal(bytes, &bundle); err != nil {
		return nil, errors.Wrapf(err, "parsing MITRE ATT&CK data loaded from %q", mitreBundleFile)
	}
	return &bundle, nil
}

// FlattenMitreMatrices flattens multiple matrices such as container, network, etc. into one enterprise matrix.
// For example,
//
//	matrix1: tactic1: technique1, technique2
//	matrix2: tactic1: technique2; tactic2: technique3
//	result:
//	  tactic1: technique1, technique2; tactic2: technique3
func FlattenMitreMatrices(matrices ...*storage.MitreAttackMatrix) []*storage.MitreAttackVector {
	tactics := make(map[string]*storage.MitreTactic)
	techniques := make(map[string]*storage.MitreTechnique)
	tacticsTechniques := make(map[string]map[string]struct{})
	for _, matrix := range matrices {
		for _, vector := range matrix.GetVectors() {
			tacticID := vector.GetTactic().GetId()
			if tactics[tacticID] == nil {
				tactics[tacticID] = vector.GetTactic()
			}

			if tacticsTechniques[tacticID] == nil {
				tacticsTechniques[tacticID] = make(map[string]struct{})
			}

			for _, technique := range vector.GetTechniques() {
				if techniques[technique.GetId()] == nil {
					techniques[technique.GetId()] = technique
				}

				if _, ok := tacticsTechniques[tacticID][technique.GetId()]; ok {
					techniques[technique.GetId()] = technique
				}
				tacticsTechniques[tacticID][technique.GetId()] = struct{}{}
			}
		}
	}

	vectors := make([]*storage.MitreAttackVector, 0, len(tactics))
	for tacticID, techniqueIDs := range tacticsTechniques {
		techniquesForTactics := make([]*storage.MitreTechnique, 0, len(techniqueIDs))
		for techniqueID := range techniqueIDs {
			techniquesForTactics = append(techniquesForTactics, techniques[techniqueID])
		}

		vectors = append(vectors, &storage.MitreAttackVector{
			Tactic:     tactics[tacticID],
			Techniques: techniquesForTactics,
		})
	}
	return vectors
}
