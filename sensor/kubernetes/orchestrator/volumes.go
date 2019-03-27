package orchestrator

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/orchestrators"
	v1 "k8s.io/api/core/v1"
)

// only hostmounts are currently supported.
func asVolumes(service *serviceWrap) (output []v1.Volume) {
	output = make([]v1.Volume, 0, len(service.Mounts)+len(service.Secrets))

	for _, m := range service.Mounts {
		output = append(output, newHostMount(m).kubernetesVolume())
	}
	for _, s := range service.Secrets {
		output = append(output, kubernetesSecretVolume(s))
	}
	return
}

func asVolumeMounts(service *serviceWrap) (output []v1.VolumeMount) {
	output = make([]v1.VolumeMount, 0, len(service.Mounts)+len(service.Secrets))

	for _, m := range service.Mounts {
		hm := newHostMount(m)
		output = append(output, hm.kubernetesVolumeMount())
	}
	for _, s := range service.Secrets {
		output = append(output, kubernetesSecretVolumeMount(s))
	}

	return
}

type hostMount struct {
	name, source, target string
}

func newHostMount(mount string) hostMount {
	split := strings.SplitN(mount, ":", 2)
	hm := hostMount{}

	hm.name = kubernetesName(split[0])
	hm.source = split[0]
	if len(split) > 1 {
		hm.target = split[1]
	} else {
		hm.target = split[0]
	}

	return hm
}

func kubernetesName(name string) string {
	replaced := invalidDNSLabelCharacter.ReplaceAllString(name, "-")
	trimmed := strings.Trim(replaced, "-")
	return strings.ToLower(trimmed)
}

func kubernetesSecretVolume(s orchestrators.Secret) v1.Volume {
	items := make([]v1.KeyToPath, 0, len(s.Items))
	for k, v := range s.Items {
		items = append(items, v1.KeyToPath{Key: k, Path: v})
	}
	return v1.Volume{
		Name: fmt.Sprintf("%s-volume", s.Name),
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName: s.Name,
				Items:      items,
			},
		},
	}
}

func kubernetesSecretVolumeMount(s orchestrators.Secret) v1.VolumeMount {
	return v1.VolumeMount{
		Name:      fmt.Sprintf("%s-volume", s.Name),
		MountPath: s.TargetPath,
	}
}

func (m hostMount) kubernetesVolume() v1.Volume {
	return v1.Volume{
		Name: m.name,
		VolumeSource: v1.VolumeSource{
			HostPath: &v1.HostPathVolumeSource{Path: m.source},
		},
	}
}

func (m hostMount) kubernetesVolumeMount() v1.VolumeMount {
	return v1.VolumeMount{
		Name:      m.name,
		MountPath: m.target,
	}
}
