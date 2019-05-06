package multipliers

import (
	"strings"

	"github.com/stackrox/rox/central/processwhitelist"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	processWhitelistHeading = `Suspicious Process Executions`

	processWhitelistSaturation = 10
	processWhitelistValue      = 4

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
		nextIncrement: float32(processWhitelistSaturation) / 5,
	}
}

func (s *scorer) addProcess() {
	s.currentScore += s.nextIncrement
	s.nextIncrement *= discountFactor
}

func (s *scorer) getScore() float32 {
	return s.currentScore
}

type processWhitelistMultiplier struct {
	whitelistGetter getters.ProcessWhitelists
	indicatorGetter getters.ProcessIndicators
}

// NewProcessWhitelists returns a multiplier for process whitelists.
func NewProcessWhitelists(whitelistGetter getters.ProcessWhitelists, indicatorGetter getters.ProcessIndicators) Multiplier {
	return &processWhitelistMultiplier{
		whitelistGetter: whitelistGetter,
		indicatorGetter: indicatorGetter,
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

func (p *processWhitelistMultiplier) Score(deployment *storage.Deployment) *storage.Risk_Result {
	scorer := newScorer()
	riskResult := &storage.Risk_Result{
		Name: processWhitelistHeading,
	}

	containerNameToWhitelistedProcesses := make(map[string]set.StringSet)
	for _, container := range deployment.GetContainers() {
		whitelist, err := p.whitelistGetter.GetProcessWhitelist(&storage.ProcessWhitelistKey{
			DeploymentId:  deployment.GetId(),
			ContainerName: container.GetName(),
		})
		if err != nil {
			log.Error(err)
			return nil
		}
		if whitelist == nil {
			continue
		}
		processSet := processwhitelist.Processes(whitelist, processwhitelist.RoxLocked)
		if processSet != nil {
			containerNameToWhitelistedProcesses[container.GetName()] = *processSet
		}
	}

	processes, err := p.indicatorGetter.SearchRawProcessIndicators(search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).ProtoQuery())
	if err != nil {
		log.Error(err)
		return nil
	}
	for _, process := range processes {
		processSet, exists := containerNameToWhitelistedProcesses[process.GetContainerName()]
		// If no explicit whitelist, then all processes are valid.
		if !exists {
			continue
		}
		if !processSet.Contains(process.GetSignal().GetName()) {
			scorer.addProcess()
			riskResult.Factors = append(riskResult.Factors, &storage.Risk_Result_Factor{
				Message: formatProcess(process),
			})
		}
	}

	if len(riskResult.GetFactors()) == 0 {
		return nil
	}

	riskResult.Score = normalizeScore(scorer.getScore(), processWhitelistSaturation, processWhitelistValue)
	return riskResult
}
