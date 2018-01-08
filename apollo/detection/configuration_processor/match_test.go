package configurationprocessor

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	t.Parallel()

	cases := []struct {
		deployment         *v1.Deployment
		policy             *v1.Policy
		expectedViolations []*v1.Alert_Violation
	}{
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					Env: &v1.ConfigurationPolicy_EnvironmentPolicy{
						Key: "Sensitive",
					},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Container Environment (key='SensitiveKey', value='SomeValue') matched configured policy (key='Sensitive')",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					Env: &v1.ConfigurationPolicy_EnvironmentPolicy{
						Key:   "Sensitive",
						Value: "^Value",
					},
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "foo", "bar"},
							User:    "root",
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					Env: &v1.ConfigurationPolicy_EnvironmentPolicy{
						Key:   "Key",
						Value: "Value",
					},
					Command: "oo",
					User:    "^root$",
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Container Configuration command=[cmd1 foo bar], user=root matched configured policy command=oo, user=^root$",
				},
				{
					Message: "Container Environment (key='Key', value='Value') matched configured policy (key='Key', value='Value')",
				},
				{
					Message: "Container Environment (key='SensitiveKey', value='SomeValue') matched configured policy (key='Key', value='Value')",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
						Volumes: []*v1.Volume{
							{
								Name:     "secret",
								Path:     "/run/secrets",
								ReadOnly: true,
								Type:     "secret",
							},
							{
								Name:     "mount",
								Path:     "/var/data",
								ReadOnly: true,
								Type:     "bind",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					VolumePolicy: &v1.ConfigurationPolicy_VolumePolicy{
						SetReadOnly: &v1.ConfigurationPolicy_VolumePolicy_ReadOnly{
							ReadOnly: true,
						},
						Type: "secret",
					},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Volume name:\"secret\" path:\"/run/secrets\" read_only:true type:\"secret\"  matched configured policy read_only=true, type=secret",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
						Volumes: []*v1.Volume{
							{
								Name:     "secret",
								Path:     "/run/secrets",
								ReadOnly: false,
								Type:     "secret",
							},
							{
								Name:     "mount",
								Path:     "/var/data",
								ReadOnly: false,
								Type:     "bind",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					VolumePolicy: &v1.ConfigurationPolicy_VolumePolicy{
						SetReadOnly: &v1.ConfigurationPolicy_VolumePolicy_ReadOnly{
							ReadOnly: true,
						},
						Type: "secret",
					},
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
						Ports: []*v1.PortConfig{
							{
								Name:          "api",
								ContainerPort: 80,
								Protocol:      "tcp",
							},
							{
								Name:          "ui",
								ContainerPort: 3000,
								Protocol:      "tcp",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					PortPolicy: &v1.ConfigurationPolicy_PortPolicy{
						Protocol: "TCP",
					},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Port name:\"api\" container_port:80 protocol:\"tcp\"  matched configured policy protocol=TCP",
				},
				{
					Message: "Port name:\"ui\" container_port:3000 protocol:\"tcp\"  matched configured policy protocol=TCP",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						Config: &v1.ContainerConfig{
							Env: []*v1.ContainerConfig_EnvironmentConfig{
								{
									Key:   "Key",
									Value: "Value",
								},
								{
									Key:   "SensitiveKey",
									Value: "SomeValue",
								},
							},
							Command: []string{"cmd1", "cmd2"},
							User:    "root",
						},
						Ports: []*v1.PortConfig{
							{
								Name:          "api",
								ContainerPort: 80,
								Protocol:      "tcp",
							},
							{
								Name:          "ui",
								ContainerPort: 3000,
								Protocol:      "tcp",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				ConfigurationPolicy: &v1.ConfigurationPolicy{
					PortPolicy: &v1.ConfigurationPolicy_PortPolicy{
						Port: 80,
					},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Port name:\"api\" container_port:80 protocol:\"tcp\"  matched configured policy port=80",
				},
			},
		},
	}

	for _, c := range cases {
		compiled, err := newCompiledConfigurationPolicy(c.policy)
		assert.NoError(t, err)

		violations := compiled.match(c.deployment)

		assert.Equal(t, c.expectedViolations, violations)
	}
}
