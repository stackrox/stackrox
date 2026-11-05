package image

import (
	"context"

	"github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Scorer is the object that encompasses the multipliers for evaluating image risk
type Scorer interface {
	// TODO(ROX-30117): Remove Score after ImageV2 model is fully rolled out
	Score(ctx context.Context, image *storage.Image) *storage.Risk
	ScoreV2(ctx context.Context, image *storage.ImageV2) *storage.Risk
}

// NewImageScorer returns a new scorer that encompasses multipliers for evaluating image risk
func NewImageScorer() Scorer {
	scoreImpl := &imageScorerImpl{
		ConfiguredMultipliers: []image.Multiplier{
			image.NewVulnerabilities(),
			image.NewRiskyComponents(),
			image.NewComponentCount(),
			image.NewImageAge(),
		},
	}

	return scoreImpl
}

type imageScorerImpl struct {
	ConfiguredMultipliers []image.Multiplier
}

// Score takes an image and evaluates its risk
func (s *imageScorerImpl) Score(ctx context.Context, image *storage.Image) *storage.Risk {
	if image.GetId() == "" {
		return nil
	}

	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	proposedScore := float32(1.0)
	legacyScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.Score(ctx, image); riskResult != nil {
			proposedScore *= riskResult.GetScore()
			legacyScore *= riskResultLegacyScore(riskResult)
			riskResults = append(riskResults, riskResult)
		}
	}

	if len(riskResults) == 0 {
		return nil
	}

	risk := &storage.Risk{
		Score:                legacyScore,
		ProposedScore:        proposedScore,
		Results:              riskResults,
		VulnerabilitySummary: vulnerabilitySummaryFromScan(image.GetScan()),
		Subject: &storage.RiskSubject{
			Id:   image.GetId(),
			Type: storage.RiskSubjectType_IMAGE,
		},
	}

	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Error(err)
		return nil
	}
	risk.Id = riskID

	return risk
}

func (s *imageScorerImpl) ScoreV2(ctx context.Context, image *storage.ImageV2) *storage.Risk {
	if image.GetId() == "" {
		return nil
	}

	riskResults := make([]*storage.Risk_Result, 0, len(s.ConfiguredMultipliers))
	proposedScore := float32(1.0)
	legacyScore := float32(1.0)
	for _, mult := range s.ConfiguredMultipliers {
		if riskResult := mult.ScoreV2(ctx, image); riskResult != nil {
			proposedScore *= riskResult.GetScore()
			legacyScore *= riskResultLegacyScore(riskResult)
			riskResults = append(riskResults, riskResult)
		}
	}

	if len(riskResults) == 0 {
		return nil
	}

	risk := &storage.Risk{
		Score:                legacyScore,
		ProposedScore:        proposedScore,
		Results:              riskResults,
		VulnerabilitySummary: vulnerabilitySummaryFromScan(image.GetScan()),
		Subject: &storage.RiskSubject{
			Id:   image.GetId(),
			Type: storage.RiskSubjectType_IMAGE,
		},
	}

	riskID, err := datastore.GetID(risk.GetSubject().GetId(), risk.GetSubject().GetType())
	if err != nil {
		log.Error(err)
		return nil
	}
	risk.Id = riskID

	return risk
}

func riskResultLegacyScore(r *storage.Risk_Result) float32 {
	if r.GetLegacyScore() > 0 {
		return r.GetLegacyScore()
	}
	return r.GetScore()
}
