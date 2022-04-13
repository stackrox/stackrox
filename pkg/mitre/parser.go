package mitre

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

const (
	subTechniqueIDSep = "."
)

// UnmarshalAndExtractMitreAttackBundle parses raw MITRE data per the MITRE specification, and extracts MITRE ATT&CK
// vectors from MITRE objects into array of `storage.MitreAttackVector`, for given MITRE domain and platform.
func UnmarshalAndExtractMitreAttackBundle(domain Domain, platform []Platform, data []byte) (*storage.MitreAttackBundle, error) {
	var rawBundle mitreBundle
	if err := json.Unmarshal(data, &rawBundle); err != nil {
		return nil, errors.Wrap(err, "parsing MITRE ATT&CK raw data")
	}
	return ExtractMitreAttackBundle(domain, platform, rawBundle.Objects)
}

// ExtractMitreAttackBundle extracts MITRE ATT&CK vectors from MITRE objects into array of `storage.MitreAttackVector`,
// for given MITRE domain and platform.
func ExtractMitreAttackBundle(domain Domain, platforms []Platform, objs []mitreObject) (*storage.MitreAttackBundle, error) {
	platformMap := make(map[Platform]struct{})
	for _, p := range platforms {
		platformMap[p] = struct{}{}
	}

	tactics := make(map[string]*storage.MitreTactic)
	// This map stores tactic short names to IDs. We need this map since the references from techniques to tactics
	// are by short names and not IDs :(
	tacticShortNameMap := make(map[string]string)
	// Collect all the tactics.
	for i := range objs {
		obj := objs[i]
		if obj.Type != tactic {
			continue
		}

		if !appliesToDomain(obj, domain) {
			continue
		}

		id := getExternalID(obj)
		if id == "" {
			continue
		}

		tactics[id] = &storage.MitreTactic{
			Id:          id,
			Name:        obj.Name,
			Description: obj.Description,
		}
		tacticShortNameMap[obj.XMitreShortname] = id
	}

	techniques := make(map[string]*storage.MitreTechnique)
	subTechiquesMap := make(map[string]struct{})
	techniquesMatrixMap := make(map[Platform]map[string]struct{})
	tacticTechniquesMap := make(map[string]map[string]struct{})
	// Collect all the techniques applicable to the platform.
	for i := range objs {
		obj := objs[i]
		// "attackPattern" represents techniques.
		if obj.Type != attackPattern {
			continue
		}

		if !appliesToDomain(obj, domain) {
			continue
		}

		matchedPlatforms, ok := appliesToAnyPlatform(obj, platformMap)
		if !ok {
			continue
		}

		techniqueID := getExternalID(obj)
		if techniqueID == "" {
			continue
		}

		techniques[techniqueID] = &storage.MitreTechnique{
			Id:          techniqueID,
			Name:        obj.Name,
			Description: obj.Description,
		}

		if obj.XMitreIsSubtechnique {
			subTechiquesMap[techniqueID] = struct{}{}
		}

		for _, platform := range matchedPlatforms {
			if techniquesMatrixMap[platform] == nil {
				techniquesMatrixMap[platform] = make(map[string]struct{})
			}
			techniquesMatrixMap[platform][techniqueID] = struct{}{}
		}

		// Obtain the reference to tactics.
		for _, phase := range obj.KillChainPhases {
			if phase.KillChainName != mitreAttackDataSrc {
				continue
			}

			tacticID := tacticShortNameMap[phase.PhaseName]
			if tacticID == "" {
				continue
			}

			if tacticTechniquesMap[tacticID] == nil {
				tacticTechniquesMap[tacticID] = make(map[string]struct{})
			}
			tacticTechniquesMap[tacticID][techniqueID] = struct{}{}
		}
	}

	// Build composite names for sub-techniques.
	for id, technique := range techniques {
		if _, ok := subTechiquesMap[id]; !ok {
			continue
		}
		pID, _ := stringutils.Split2(id, subTechniqueIDSep)
		if pID == "" {
			return nil, errors.Errorf("MITRE ATT&CK sub-technique ID %s does not contain technique ID", id)
		}
		pTechnique := techniques[pID]
		if pTechnique == nil {
			return nil, errors.Errorf("MITRE ATT&CK technique %s not found", pID)
		}
		technique.Name = fmt.Sprintf("%s: %s", pTechnique.GetName(), technique.GetName())
	}

	var version string
	for i := range objs {
		obj := objs[i]
		if obj.Type != metadata {
			continue
		}
		version = obj.Version
	}

	// Build full vectors.
	vectors := buildVectors(tactics, techniques, tacticTechniquesMap)
	// Build bundles.
	return generateBundle(version, domain, techniquesMatrixMap, vectors...), nil
}

func buildVectors(
	tactics map[string]*storage.MitreTactic,
	techniques map[string]*storage.MitreTechnique,
	tacticTechniquesMap map[string]map[string]struct{},
) []*storage.MitreAttackVector {
	var vectors []*storage.MitreAttackVector
	for tacticID, techniquesMap := range tacticTechniquesMap {
		vector := &storage.MitreAttackVector{
			Tactic: tactics[tacticID],
		}

		for techniqueID := range techniquesMap {
			vector.Techniques = append(vector.Techniques, techniques[techniqueID])
		}
		vectors = append(vectors, vector)

		sort.SliceStable(vector.Techniques, func(i, j int) bool {
			return vector.Techniques[i].GetId() < vector.Techniques[j].GetId()
		})
	}

	sort.SliceStable(vectors, func(i, j int) bool {
		return vectors[i].GetTactic().GetId() < vectors[j].GetTactic().GetId()
	})
	return vectors
}

// Builds out vectors into MITRE ATT&CK matrix per Domain+Platform applicability.
func generateBundle(
	version string,
	domain Domain,
	techniqueMatrix map[Platform]map[string]struct{},
	vectors ...*storage.MitreAttackVector,
) *storage.MitreAttackBundle {
	bundle := &storage.MitreAttackBundle{
		Version: version,
	}
	for platform, techniqueIDs := range techniqueMatrix {
		var filteredVectors []*storage.MitreAttackVector
		for _, vector := range vectors {
			var filteredTechniques []*storage.MitreTechnique
			for _, technique := range vector.GetTechniques() {
				if _, ok := techniqueIDs[technique.GetId()]; ok {
					filteredTechniques = append(filteredTechniques, technique)
				}
			}

			if len(filteredTechniques) == 0 {
				continue
			}

			filteredVectors = append(filteredVectors, &storage.MitreAttackVector{
				Tactic:     vector.GetTactic(),
				Techniques: filteredTechniques,
			})
		}

		if len(filteredVectors) == 0 {
			continue
		}

		bundle.Matrices = append(bundle.Matrices, &storage.MitreAttackMatrix{
			MatrixInfo: &storage.MitreAttackMatrix_MatrixInfo{
				Domain:   domain.String(),
				Platform: platform.String(),
			},
			Vectors: filteredVectors,
		})
	}

	sort.SliceStable(bundle.Matrices, func(i, j int) bool {
		return bundle.Matrices[i].GetMatrixInfo().GetPlatform() < bundle.Matrices[j].GetMatrixInfo().GetPlatform()
	})
	return bundle
}

func appliesToDomain(mitreObj mitreObject, domain Domain) bool {
	for _, d := range mitreObj.XMitreDomains {
		if d == domain {
			return true
		}
	}
	return false
}

func appliesToAnyPlatform(mitreObj mitreObject, platforms map[Platform]struct{}) ([]Platform, bool) {
	var ret []Platform
	for _, p := range mitreObj.XMitrePlatforms {
		if _, ok := platforms[p]; ok {
			ret = append(ret, p)
		}
	}
	return ret, len(ret) > 0
}

func getExternalID(mitreObj mitreObject) string {
	for _, extRef := range mitreObj.ExternalReferences {
		if extRef.SourceName == mitreAttackDataSrc {
			return extRef.ExternalID
		}
	}
	return ""
}
