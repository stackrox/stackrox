package risk

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	"github.com/deckarep/golang-set"
)

const (
	serviceConfigHeading = "Service Configuration"
	maxScore             = 8
)

// ServiceConfigMultiplier is a scorer for the service configuration
type ServiceConfigMultiplier struct{}

// NewServiceConfigMultiplier scores the data based on the service configuration
func NewServiceConfigMultiplier() *ServiceConfigMultiplier {
	return &ServiceConfigMultiplier{}
}

func (s *ServiceConfigMultiplier) scoreVolumesAndSecrets(deployment *v1.Deployment) (volumeFactor, secretFactor string) {
	var volumeNames []string
	var secrets []string
	for _, container := range deployment.GetContainers() {
		for _, volume := range container.GetVolumes() {
			if volume.GetType() == "secret" {
				secrets = append(secrets, volume.GetName())
			} else if !volume.GetReadOnly() {
				volumeNames = append(volumeNames, volume.GetName())
			}
		}
	}
	if len(volumeNames) != 0 {
		volumeFactor = fmt.Sprintf("Volumes %s were mounted RW", strings.Join(volumeNames, ", "))
	}
	if len(secrets) != 0 {
		secretFactor = fmt.Sprintf("Secrets %s are used inside the deployment", strings.Join(secrets, ", "))
	}
	return
}

var capAdds = map[string]struct{}{
	"ALL":            {},
	"CAP_SYS_ADMIN":  {},
	"CAP_NET_ADMIN":  {},
	"CAP_SYS_MODULE": {},
}

func (s *ServiceConfigMultiplier) scoreCapabilities(deployment *v1.Deployment) (capAddFactor, capDropFactor string) {
	capsAdded := mapset.NewSet()
	capsDropped := mapset.NewSet()
	for _, container := range deployment.GetContainers() {
		context := container.GetSecurityContext()
		for _, cap := range context.GetAddCapabilities() {
			if _, ok := capAdds[cap]; ok {
				capsAdded.Add(cap)
			}
		}
		for _, cap := range context.GetDropCapabilities() {
			capsDropped.Add(cap)
		}
	}
	if capsAdded.Cardinality() != 0 {
		addedSlice := set.StringSliceFromSet(capsAdded)
		capAddFactor = fmt.Sprintf("Capabilities %s were added", strings.Join(addedSlice, ", "))
	}
	if capsDropped.Cardinality() == 0 {
		capDropFactor = fmt.Sprintf("No capabilities were dropped")
	}
	return
}

func (s *ServiceConfigMultiplier) scorePrivilege(deployment *v1.Deployment) string {
	for _, container := range deployment.GetContainers() {
		if container.GetSecurityContext().GetPrivileged() {
			return "A container in the deployment is privileged"
		}
	}
	return ""
}

// Score takes a deployment and evaluates its risk based on the service configuration
func (s *ServiceConfigMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	riskResult := &v1.Risk_Result{
		Name: serviceConfigHeading,
	}
	var overallScore float32
	volumeFactor, secretFactor := s.scoreVolumesAndSecrets(deployment)
	if volumeFactor != "" {
		overallScore++
		riskResult.Factors = append(riskResult.Factors, volumeFactor)
	}
	if secretFactor != "" {
		overallScore++
		riskResult.Factors = append(riskResult.Factors, secretFactor)
	}
	capAddFactor, capDropFactor := s.scoreCapabilities(deployment)
	if capAddFactor != "" {
		overallScore++
		riskResult.Factors = append(riskResult.Factors, capAddFactor)
	}
	if capDropFactor != "" {
		overallScore++
		riskResult.Factors = append(riskResult.Factors, capDropFactor)
	}
	if factor := s.scorePrivilege(deployment); factor != "" {
		overallScore *= 2
		if overallScore < 2 {
			overallScore = 2
		}
		riskResult.Factors = append(riskResult.Factors, factor)
	}
	// riskResult.Score is the normalized [1.0,2.0] score
	riskResult.Score = (overallScore / maxScore) + 1
	return riskResult
}
