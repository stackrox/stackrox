package deployment

import (
	"context"
	"strings"

	"github.com/stackrox/stackrox/central/processbaseline/evaluator"
	"github.com/stackrox/stackrox/central/risk/multipliers"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/stringutils"
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

func formatProcess(process *storage.ProcessIndicator) string {
	sb := strings.Builder{}
	stringutils.WriteStringf(&sb, "Detected execution of suspicious process %q", process.GetSignal().GetName())
	if len(process.GetSignal().GetArgs()) > 0 {
		stringutils.WriteStringf(&sb, " with args %q", process.GetSignal().GetArgs())
	}
	stringutils.WriteStrings(&sb, " in container ", process.GetContainerName())
	return sb.String()
}

func (p *processBaselineMultiplier) Score(_ context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	violatingProcesses, err := p.evaluator.EvaluateBaselinesAndPersistResult(deployment)
	if err != nil {
		log.Errorf("Couldn't evaluate process baseline: %v", err)
		return nil
	}

	scorer := newScorer()
	riskResult := &storage.Risk_Result{
		Name: processBaselineHeading,
	}

	for _, process := range violatingProcesses {
		scorer.addProcess()
		riskResult.Factors = append(riskResult.Factors, &storage.Risk_Result_Factor{
			Message: formatProcess(process),
		})
	}

	if len(riskResult.GetFactors()) == 0 {
		return nil
	}

	riskResult.Score = multipliers.NormalizeScore(scorer.getScore(), processBaselineSaturation, processBaselineValue)
	return riskResult
}
