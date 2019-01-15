package orchestrator

import (
	"testing"

	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertDeployment(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    *serviceWrap
		expected *v1beta1.Deployment
	}{
		{
			input: &serviceWrap{
				SystemService: orchestrators.SystemService{
					Global:  false,
					Command: []string{"start", "--flag"},
					Mounts:  []string{"/var/run/docker.sock:/var/run/docker.sock"},
					Envs:    []string{"hello=world", "foo=bar"},
					Name:    `test`,
					Image:   `stackrox/test:latest`,
				},
				namespace: "stackrox",
			},
			expected: &v1beta1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       `Deployment`,
					APIVersion: `extensions/v1beta1`,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      `test`,
					Namespace: `stackrox`,
				},
				Spec: v1beta1.DeploymentSpec{
					Replicas: &[]int32{1}[0],
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: `stackrox`,
							Labels: map[string]string{
								`com.docker.stack.namespace`: `prevent`,
								`com.prevent.service-name`:   `test`,
							},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Command: []string{`start`, `--flag`},
									Name:    `test`,
									Env: []v1.EnvVar{
										{
											Name:  `hello`,
											Value: `world`,
										},
										{
											Name:  `foo`,
											Value: `bar`,
										},
									},
									Image: `stackrox/test:latest`,
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      `var-run-docker-sock`,
											MountPath: `/var/run/docker.sock`,
										},
									},
								},
							},
							RestartPolicy: v1.RestartPolicyAlways,
							Volumes: []v1.Volume{
								{
									Name: `var-run-docker-sock`,
									VolumeSource: v1.VolumeSource{
										HostPath: &v1.HostPathVolumeSource{
											Path: `/var/run/docker.sock`,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	convert := &converter{}

	for _, c := range cases {
		actual := convert.asDeployment(c.input)

		assert.Equal(t, c.expected, actual)
	}
}

func TestCovertDaemonSet(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input    *serviceWrap
		expected *v1beta1.DaemonSet
	}{
		{
			input: &serviceWrap{
				SystemService: orchestrators.SystemService{
					Global: true,
					Mounts: []string{"/var/run/docker.sock:/var/run/docker.sock", "/tmp:/var/lib"},
					Envs:   []string{"hello=world", "foo=bar"},
					Name:   `daemon`,
					Image:  `stackrox/daemon:1.0`,
				},
				namespace: "prevent",
			},
			expected: &v1beta1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       `DaemonSet`,
					APIVersion: `extensions/v1beta1`,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      `daemon`,
					Namespace: `prevent`,
				},
				Spec: v1beta1.DaemonSetSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: `prevent`,
							Labels: map[string]string{
								`com.docker.stack.namespace`: `prevent`,
								`com.prevent.service-name`:   `daemon`,
							},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name: `daemon`,
									Env: []v1.EnvVar{
										{
											Name:  `hello`,
											Value: `world`,
										},
										{
											Name:  `foo`,
											Value: `bar`,
										},
									},
									Image: `stackrox/daemon:1.0`,
									VolumeMounts: []v1.VolumeMount{
										{
											Name:      `var-run-docker-sock`,
											MountPath: `/var/run/docker.sock`,
										},
										{
											Name:      `tmp`,
											MountPath: `/var/lib`,
										},
									},
								},
							},
							Tolerations: []v1.Toleration{
								{
									Effect:   v1.TaintEffectNoSchedule,
									Operator: v1.TolerationOpExists,
								},
							},
							RestartPolicy: v1.RestartPolicyAlways,
							Volumes: []v1.Volume{
								{
									Name: `var-run-docker-sock`,
									VolumeSource: v1.VolumeSource{
										HostPath: &v1.HostPathVolumeSource{
											Path: `/var/run/docker.sock`,
										},
									},
								},
								{
									Name: `tmp`,
									VolumeSource: v1.VolumeSource{
										HostPath: &v1.HostPathVolumeSource{
											Path: `/tmp`,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	convert := &converter{}

	for _, c := range cases {
		actual := convert.asDaemonSet(c.input)

		assert.Equal(t, c.expected, actual)
	}

}
