package orchestrator

import (
	"fmt"

	"github.com/stackrox/rox/pkg/orchestrators"
	v1 "k8s.io/api/core/v1"
)

func convertSpecialEnvs(envVars []orchestrators.SpecialEnvVar) []v1.EnvVar {
	result := make([]v1.EnvVar, 0, len(envVars))

	for _, envVar := range envVars {
		specialEnvVar, err := convertSpecialEnv(envVar)
		if err != nil {
			log.Errorf("Failed to convert special environment variable: %v", err)
			continue
		}
		result = append(result, specialEnvVar)
	}
	return result
}

func convertSpecialEnv(envVar orchestrators.SpecialEnvVar) (v1.EnvVar, error) {
	src, err := specialEnvVarSrc(envVar)
	if err != nil {
		return v1.EnvVar{}, err
	}
	return v1.EnvVar{
		Name:      string(envVar),
		ValueFrom: src,
	}, nil
}

func specialEnvVarSrc(envVar orchestrators.SpecialEnvVar) (*v1.EnvVarSource, error) {
	switch envVar {
	case orchestrators.NodeName:
		return &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: "spec.nodeName",
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported special environment variable %s", envVar)
	}
}
