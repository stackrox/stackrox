package privilegeprocessor

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
						SecurityContext: &v1.SecurityContext{
							Privileged:       true,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							Privileged:       true,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Container privileged set to true matched configured policy",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							Privileged:       false,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: false,
					},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Container privileged set to false matched configured policy",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							Privileged:       true,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
					AddCapabilities: []string{"CAP_IPC_LOCK"},
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							Privileged:       true,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
					AddCapabilities: []string{"CAP_SYS_ADMIN"},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Container privileged set to true matched configured policy",
				},
				{
					Message: "Container with add capabilities [CAP_SYS_ADMIN CAP_SYS_MODULE] matches policy [CAP_SYS_ADMIN]",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							Privileged:       true,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
					AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
					DropCapabilities: []string{"CAP_KILL"},
				},
			},
			expectedViolations: []*v1.Alert_Violation{
				{
					Message: "Container privileged set to true matched configured policy",
				},
				{
					Message: "Container with add capabilities [CAP_SYS_ADMIN CAP_SYS_MODULE] matches policy [CAP_SYS_ADMIN CAP_SYS_MODULE]",
				},
				{
					Message: "Container with drop capabilities [CAP_CHOWN] did not contain all configured drop capabilities [CAP_KILL]",
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							Privileged:       true,
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
					{
						SecurityContext: &v1.SecurityContext{
							AddCapabilities: []string{"CAP_SYS_ADMIN", "CAP_IPC_LOCK"},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
					AddCapabilities: []string{"CAP_SYS_ADMIN", "CAP_IPC_LOCK", "CAP_SYS_MODULE"},
				},
			},
		},
		{
			deployment: &v1.Deployment{
				Containers: []*v1.Container{
					{
						SecurityContext: &v1.SecurityContext{
							AddCapabilities:  []string{"CAP_SYS_ADMIN", "CAP_SYS_MODULE"},
							DropCapabilities: []string{"CAP_CHOWN"},
							Selinux: &v1.SecurityContext_SELinux{
								User:  "user",
								Role:  "role",
								Type:  "type",
								Level: "level",
							},
						},
					},
				},
			},
			policy: &v1.Policy{
				PrivilegePolicy: &v1.PrivilegePolicy{
					SetPrivileged: &v1.PrivilegePolicy_Privileged{
						Privileged: true,
					},
				},
			},
		},
	}

	for _, c := range cases {
		compiled, exist, err := NewCompiledPrivilegePolicy(c.policy)
		assert.True(t, exist)
		assert.NoError(t, err)

		var violations []*v1.Alert_Violation
		for _, container := range c.deployment.GetContainers() {
			vs := compiled.Match(c.deployment, container)
			violations = append(violations, vs...)
		}

		assert.Equal(t, c.expectedViolations, violations)
	}
}
