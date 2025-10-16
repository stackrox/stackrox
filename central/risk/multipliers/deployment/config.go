package deployment

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

const (
	// ServiceConfigHeading is the risk result name for scores calculated by this multiplier.
	ServiceConfigHeading = "Service Configuration"

	configSaturation = 8
	configMaxScore   = 2
)

// serviceConfigMultiplier is a scorer for the service configuration
type serviceConfigMultiplier struct{}

// NewServiceConfig scores the data based on the service configuration
func NewServiceConfig() Multiplier {
	return &serviceConfigMultiplier{}
}

// Score takes a deployment and evaluates its risk based on the service configuration
func (s *serviceConfigMultiplier) Score(_ context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	riskResult := &storage.Risk_Result{}
	riskResult.SetName(ServiceConfigHeading)
	var overallScore float32
	if volumeFactor := s.scoreVolumes(deployment); volumeFactor != "" {
		overallScore++
		rrf := &storage.Risk_Result_Factor{}
		rrf.SetMessage(volumeFactor)
		riskResult.SetFactors(append(riskResult.GetFactors(), rrf))
	}
	if secretFactor := s.scoreSecrets(deployment); secretFactor != "" {
		overallScore++
		rrf := &storage.Risk_Result_Factor{}
		rrf.SetMessage(secretFactor)
		riskResult.SetFactors(append(riskResult.GetFactors(), rrf))
	}
	capAddFactor, capDropFactor := s.scoreCapabilities(deployment)
	if capAddFactor != "" {
		overallScore++
		rrf := &storage.Risk_Result_Factor{}
		rrf.SetMessage(capAddFactor)
		riskResult.SetFactors(append(riskResult.GetFactors(), rrf))
	}
	if capDropFactor != "" {
		overallScore++
		rrf := &storage.Risk_Result_Factor{}
		rrf.SetMessage(capDropFactor)
		riskResult.SetFactors(append(riskResult.GetFactors(), rrf))
	}
	if factor := s.scorePrivilege(deployment); factor != "" {
		overallScore *= 2
		rrf := &storage.Risk_Result_Factor{}
		rrf.SetMessage(factor)
		riskResult.SetFactors(append(riskResult.GetFactors(), rrf))
	}
	if len(riskResult.GetFactors()) == 0 {
		return nil
	}
	// riskResult.Score is the normalized [1.0,2.0] score
	riskResult.SetScore(multipliers.NormalizeScore(overallScore, configSaturation, configMaxScore))
	return riskResult
}

func (s *serviceConfigMultiplier) scoreVolumes(deployment *storage.Deployment) string {
	var volumeNames []string
	for _, container := range deployment.GetContainers() {
		for _, volume := range container.GetVolumes() {
			if !volume.GetReadOnly() {
				volumeNames = append(volumeNames, volume.GetName())
			}
		}
	}
	if len(volumeNames) != 0 {
		return fmt.Sprintf("Volumes %s were mounted RW", strings.Join(volumeNames, ", "))
	}
	return ""
}

func (s *serviceConfigMultiplier) scoreSecrets(deployment *storage.Deployment) string {
	var secrets []string
	for _, container := range deployment.GetContainers() {
		for _, secret := range container.GetSecrets() {
			secrets = append(secrets, secret.GetName())
		}
	}
	if len(secrets) != 0 {
		return fmt.Sprintf("Secrets %s are used inside the deployment", strings.Join(secrets, ", "))
	}
	return ""
}

var relevantCapAdds = set.NewFrozenStringSet(
	"ALL",
	"SYS_ADMIN",
	"NET_ADMIN",
	"SYS_MODULE",
)

func (s *serviceConfigMultiplier) scoreCapabilities(deployment *storage.Deployment) (capAddFactor, capDropFactor string) {
	capsAdded := set.NewStringSet()
	capsDropped := set.NewStringSet()
	for _, container := range deployment.GetContainers() {
		context := container.GetSecurityContext()
		for _, cap := range context.GetAddCapabilities() {
			if relevantCapAdds.Contains(strings.ToUpper(cap)) {
				capsAdded.Add(cap)
			}
		}
		for _, cap := range context.GetDropCapabilities() {
			capsDropped.Add(cap)
		}
	}
	if !capsAdded.IsEmpty() {
		capAddFactor = fmt.Sprintf("Capabilities %s were added", capsAdded.ElementsString(", "))
	}
	if capsDropped.IsEmpty() {
		capDropFactor = "No capabilities were dropped"
	}
	return
}

func (s *serviceConfigMultiplier) scorePrivilege(deployment *storage.Deployment) string {
	for _, container := range deployment.GetContainers() {
		if container.GetSecurityContext().GetPrivileged() {
			return fmt.Sprintf("Container %q in the deployment is privileged", container.GetName())
		}
	}
	return ""
}
