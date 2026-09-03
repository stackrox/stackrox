package service

import (
	"encoding/json"

	"github.com/stackrox/rox/generated/storage"
)

// buildSanitizedRiskContext produces a minimal JSON representation of the
// deployment and risk data suitable for sending to an external LLM. It keeps
// only fields relevant to risk analysis and strips sensitive data (env vars,
// secret paths) and wasteful metadata (IDs, hashes, empty fields).
func buildSanitizedRiskContext(deployment *storage.Deployment, risk *storage.Risk) (string, error) {
	ctx := sanitizedContext{
		Deployment: sanitizeDeployment(deployment),
		Risk:       sanitizeRisk(risk),
	}
	data, err := json.Marshal(ctx)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type sanitizedContext struct {
	Deployment sanitizedDeployment `json:"deployment"`
	Risk       sanitizedRisk       `json:"risk"`
}

type sanitizedDeployment struct {
	Name                          string               `json:"name"`
	Namespace                     string               `json:"namespace"`
	ClusterName                   string               `json:"clusterName"`
	Type                          string               `json:"type"`
	Created                       string               `json:"created,omitempty"`
	ServiceAccount                string               `json:"serviceAccount,omitempty"`
	ServiceAccountPermissionLevel string               `json:"serviceAccountPermissionLevel,omitempty"`
	AutomountServiceAccountToken  bool                 `json:"automountServiceAccountToken"`
	HostNetwork                   bool                 `json:"hostNetwork"`
	HostPid                       bool                 `json:"hostPid"`
	HostIpc                       bool                 `json:"hostIpc"`
	Containers                    []sanitizedContainer `json:"containers"`
}

type sanitizedContainer struct {
	Name            string                   `json:"name"`
	ImageFullName   string                   `json:"imageFullName,omitempty"`
	SecurityContext *storage.SecurityContext `json:"securityContext,omitempty"`
	Resources       *storage.Resources       `json:"resources,omitempty"`
	UID             int64                    `json:"uid,omitempty"`
	LivenessProbe   *storage.LivenessProbe   `json:"livenessProbe,omitempty"`
	ReadinessProbe  *storage.ReadinessProbe  `json:"readinessProbe,omitempty"`
}

type sanitizedRisk struct {
	Score   float32               `json:"score"`
	Results []sanitizedRiskResult `json:"results,omitempty"`
}

type sanitizedRiskResult struct {
	Name    string                `json:"name"`
	Score   float32               `json:"score"`
	Factors []sanitizedRiskFactor `json:"factors,omitempty"`
}

type sanitizedRiskFactor struct {
	Message string `json:"message"`
}

func sanitizeDeployment(d *storage.Deployment) sanitizedDeployment {
	sd := sanitizedDeployment{
		Name:                         d.GetName(),
		Namespace:                    d.GetNamespace(),
		ClusterName:                  d.GetClusterName(),
		Type:                         d.GetType(),
		ServiceAccount:               d.GetServiceAccount(),
		AutomountServiceAccountToken: d.GetAutomountServiceAccountToken(),
		HostNetwork:                  d.GetHostNetwork(),
		HostPid:                      d.GetHostPid(),
		HostIpc:                      d.GetHostIpc(),
	}

	if d.GetServiceAccountPermissionLevel() != storage.PermissionLevel_UNSET {
		sd.ServiceAccountPermissionLevel = d.GetServiceAccountPermissionLevel().String()
	}

	if d.GetCreated() != nil {
		sd.Created = d.GetCreated().AsTime().Format("2006-01-02T15:04:05Z")
	}

	for _, c := range d.GetContainers() {
		sc := sanitizedContainer{
			Name:            c.GetName(),
			SecurityContext: c.GetSecurityContext(),
			UID:             c.GetConfig().GetUid(),
		}

		if img := c.GetImage(); img != nil && img.GetName() != nil {
			sc.ImageFullName = img.GetName().GetFullName()
		}

		if res := c.GetResources(); res != nil {
			sc.Resources = res
		}

		sc.LivenessProbe = c.GetLivenessProbe()
		sc.ReadinessProbe = c.GetReadinessProbe()

		sd.Containers = append(sd.Containers, sc)
	}
	return sd
}

func sanitizeRisk(r *storage.Risk) sanitizedRisk {
	if r == nil {
		return sanitizedRisk{}
	}
	sr := sanitizedRisk{
		Score: r.GetScore(),
	}
	for _, result := range r.GetResults() {
		srr := sanitizedRiskResult{
			Name:  result.GetName(),
			Score: result.GetScore(),
		}
		for _, factor := range result.GetFactors() {
			if factor.GetMessage() != "" {
				srr.Factors = append(srr.Factors, sanitizedRiskFactor{
					Message: factor.GetMessage(),
				})
			}
		}
		sr.Results = append(sr.Results, srr)
	}
	return sr
}
