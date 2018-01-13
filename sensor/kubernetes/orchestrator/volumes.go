package orchestrator

import (
	"strings"

	"k8s.io/api/core/v1"
)

// only hostmounts are currently supported.
func (c converter) asVolumes(service *serviceWrap) (output []v1.Volume) {
	output = make([]v1.Volume, len(service.Mounts))

	for i, m := range service.Mounts {
		hm := newHostMount(m)
		output[i] = hm.kubernetesVolume()
	}

	return
}

func (c converter) asVolumeMounts(service *serviceWrap) (output []v1.VolumeMount) {
	output = make([]v1.VolumeMount, len(service.Mounts))

	for i, m := range service.Mounts {
		hm := newHostMount(m)
		output[i] = hm.kubernetesVolumeMount()
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
