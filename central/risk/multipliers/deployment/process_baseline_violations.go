package deployment

import (
	"context"
	"strings"

	"github.com/stackrox/rox/central/processbaseline/evaluator"
	"github.com/stackrox/rox/central/processindicator/views"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	processBaselineHeading = `Suspicious Process Executions`

	processBaselineSaturation = 10
	processBaselineValue      = 4

	discountFactor = 0.9
)

// The scorer abstracts the scoring logic.
// Use newScorer() to initialize a scorer.
// Every time we see a new process, the scorer adds an increment to the current score;
// the more processes we see, the lower the increment becomes.
type scorer struct {
	currentScore  float32
	nextIncrement float32
}

func newScorer() *scorer {
	return &scorer{
		nextIncrement: float32(processBaselineSaturation) / 5,
	}
}

func (s *scorer) addProcess() {
	s.currentScore += s.nextIncrement
	s.nextIncrement *= discountFactor
}

func (s *scorer) getScore() float32 {
	return s.currentScore
}

type processBaselineMultiplier struct {
	evaluator evaluator.Evaluator
}

// NewProcessBaselines returns a multiplier for process baselines.
func NewProcessBaselines(evaluator evaluator.Evaluator) Multiplier {
	return &processBaselineMultiplier{
		evaluator: evaluator,
	}
}

func formatProcess(process *views.ProcessIndicatorRiskView) string {
	sb := strings.Builder{}
	stringutils.WriteStringf(&sb, "Detected execution of suspicious process %q", process.SignalName)
	if len(process.SignalArgs) > 0 {
		stringutils.WriteStringf(&sb, " with args %q", process.SignalArgs)
	}
	stringutils.WriteStrings(&sb, " in container ", process.ContainerName)
	return sb.String()
}

func (p *processBaselineMultiplier) Score(_ context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	if !env.ProcessBaselineRisk.BooleanSetting() {
		return nil
	}
	violatingProcesses, err := p.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	if err != nil {
		log.Errorf("Couldn't evaluate process baseline: %v", err)
		return nil
	}

	scorer := newScorer()
	riskResult := &storage.Risk_Result{}
	riskResult.SetName(processBaselineHeading)

	for _, process := range violatingProcesses {
		scorer.addProcess()
		rrf := &storage.Risk_Result_Factor{}
		rrf.SetMessage(formatProcess(process))
		riskResult.SetFactors(append(riskResult.GetFactors(), rrf))
	}

	if len(riskResult.GetFactors()) == 0 {
		return nil
	}

	riskResult.SetScore(multipliers.NormalizeScore(scorer.getScore(), processBaselineSaturation, processBaselineValue))
	return riskResult
}
