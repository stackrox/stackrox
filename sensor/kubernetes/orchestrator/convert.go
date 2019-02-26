package orchestrator

import (
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/orchestrators"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	appLabel = `app`
)

var (
	invalidDNSLabelCharacter = regexp.MustCompile("[^A-Za-z0-9-]")
)

type serviceWrap struct {
	orchestrators.SystemService
	namespace   string
	tolerations []v1.Toleration
}

func asDaemonSet(service *serviceWrap) *v1beta1.DaemonSet {
	service.tolerations = []v1.Toleration{
		{
			Effect:   v1.TaintEffectNoSchedule,
			Operator: v1.TolerationOpExists,
		},
	}
	return &v1beta1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: objectMeta(service),
		Spec: v1beta1.DaemonSetSpec{
			Template: asKubernetesPod(service),
		},
	}
}

func asDeployment(service *serviceWrap) *v1beta1.Deployment {
	replicas := int32(1)
	return &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: objectMeta(service),
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: asKubernetesPod(service),
		},
	}
}

func objectMeta(service *serviceWrap) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:         service.Name,
		Namespace:    service.namespace,
		GenerateName: service.GenerateName,
		Labels:       deploymentLabels(service),
	}
}

func asKubernetesPod(service *serviceWrap) v1.PodTemplateSpec {
	return v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: service.namespace,
			Labels:    podLabels(service),
		},
		Spec: v1.PodSpec{
			Containers:         asContainers(service),
			ServiceAccountName: service.ServiceAccount,
			RestartPolicy:      v1.RestartPolicyAlways,
			Volumes:            asVolumes(service),
			HostPID:            service.HostPID,
			Tolerations:        service.tolerations,
		},
	}
}

func deploymentLabels(service *serviceWrap) (labels map[string]string) {
	labels = make(map[string]string)

	name := service.Name
	if name == "" {
		name = service.GenerateName
	}
	labels[appLabel] = name
	return
}

func podLabels(service *serviceWrap) (labels map[string]string) {
	labels = deploymentLabels(service)
	for k, v := range service.ExtraPodLabels {
		labels[k] = v
	}
	return
}

func allEnvs(service *serviceWrap) []v1.EnvVar {
	allEnvs := make([]v1.EnvVar, 0, len(service.Envs)+len(service.SpecialEnvs))
	allEnvs = append(allEnvs, convertSpecialEnvs(service.SpecialEnvs)...)
	allEnvs = append(allEnvs, asEnv(service.Envs)...)
	return allEnvs
}

func asContainers(service *serviceWrap) []v1.Container {
	containerName := service.Name
	if containerName == "" {
		containerName = service.GenerateName
	}
	return []v1.Container{
		{
			Name:         containerName,
			Env:          allEnvs(service),
			Image:        service.Image,
			Command:      service.Command,
			VolumeMounts: asVolumeMounts(service),
		},
	}
}

func asEnv(envs []string) (vars []v1.EnvVar) {
	for _, env := range envs {
		split := strings.SplitN(env, "=", 2)
		if len(split) == 2 {
			vars = append(vars, v1.EnvVar{
				Name:  split[0],
				Value: split[1],
			})
		}
	}

	return
}
