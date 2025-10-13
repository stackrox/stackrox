package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationFromRawBytes(t *testing.T) {
	cases := map[string]struct {
		rawConfigurations      [][]byte
		expectedConfigurations []Configuration
		fail                   bool
	}{
		"invalid access scope shouldn't be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
rules:
  included:
    - cluster: clusterA
      namespaces:
      - namespaceA1
  clusterLabelSelectors:
    - requirements:
      - key: a
        operator: BOGUS
        values: [a, b, c]
`),
			},
			fail: true,
		},
		"single raw role configuration should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
`),
			},
			expectedConfigurations: []Configuration{
				&Role{
					Name:          "test-name",
					Description:   "test-description",
					AccessScope:   "access-scope",
					PermissionSet: "permission-set",
				},
			},
		},
		"single raw access scope configuration should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
rules:
  included:
    - cluster: clusterA
      namespaces:
      - namespaceA1
  clusterLabelSelectors:
    - requirements:
      - key: a
        operator: IN
        values: [a, b, c]
`),
			},
			expectedConfigurations: []Configuration{
				&AccessScope{
					Name:        "test-name",
					Description: "test-description",
					Rules: Rules{
						IncludedObjects: []IncludedObject{
							{
								Cluster:    "clusterA",
								Namespaces: []string{"namespaceA1"},
							},
						},
						ClusterLabelSelectors: []LabelSelector{
							{
								Requirements: []Requirement{
									{
										Key:      "a",
										Operator: Operator(storage.LabelSelector_IN),
										Values:   []string{"a", "b", "c"},
									},
								},
							},
						},
					},
				},
			},
		},
		"single raw auth provider should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
minimumRole: "None"
uiEndpoint: "https://localhost:8000"
groups:
- key: "email"
  value: "admin@stackrox.com"
  role: "Admin"
oidc:
  issuer: "https://stackrox.com"
  mode: "auto select"
  clientID: "some-client-id"
  clientSecret: "some-client-secret"
  disableOfflineAccessScope: true
`),
			},
			expectedConfigurations: []Configuration{
				&AuthProvider{
					Name:            "test-name",
					MinimumRoleName: "None",
					UIEndpoint:      "https://localhost:8000",
					Groups: []Group{
						{
							AttributeKey:   "email",
							AttributeValue: "admin@stackrox.com",
							RoleName:       "Admin",
						},
					},
					OIDCConfig: &OIDCConfig{
						Issuer:                    "https://stackrox.com",
						CallbackMode:              "auto select",
						ClientID:                  "some-client-id",
						ClientSecret:              "some-client-secret",
						DisableOfflineAccessScope: true,
					},
				},
			},
		},
		"single permission set should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
resources:
- resource: a
  access: READ_ACCESS
- resource: b
  access: READ_WRITE_ACCESS
`),
			},
			expectedConfigurations: []Configuration{
				&PermissionSet{
					Name:        "test-name",
					Description: "test-description",
					Resources: []ResourceWithAccess{
						{
							Resource: "a",
							Access:   Access(storage.Access_READ_ACCESS),
						},
						{
							Resource: "b",
							Access:   Access(storage.Access_READ_WRITE_ACCESS),
						},
					},
				},
			},
		},
		"single multi-line configuration of roles should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
---
name: another-test-name
description: another-test-description
accessScope: another-access-scope
permissionSet: another-permission-set
`),
			},
			expectedConfigurations: []Configuration{
				&Role{
					Name:          "test-name",
					Description:   "test-description",
					AccessScope:   "access-scope",
					PermissionSet: "permission-set",
				},
				&Role{
					Name:          "another-test-name",
					Description:   "another-test-description",
					AccessScope:   "another-access-scope",
					PermissionSet: "another-permission-set",
				},
			},
		},
		"single multi-line configuration of roles w/ trailing delimiter should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
---
name: another-test-name
description: another-test-description
accessScope: another-access-scope
permissionSet: another-permission-set
---
`),
			},
			expectedConfigurations: []Configuration{
				&Role{
					Name:          "test-name",
					Description:   "test-description",
					AccessScope:   "access-scope",
					PermissionSet: "permission-set",
				},
				&Role{
					Name:          "another-test-name",
					Description:   "another-test-description",
					AccessScope:   "another-access-scope",
					PermissionSet: "another-permission-set",
				},
			},
		},
		"multiple raw role configurations should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
`),
				[]byte(`
name: another-test-name
description: another-test-description
accessScope: another-access-scope
permissionSet: another-permission-set
`),
			},
			expectedConfigurations: []Configuration{
				&Role{
					Name:          "test-name",
					Description:   "test-description",
					AccessScope:   "access-scope",
					PermissionSet: "permission-set",
				},
				&Role{
					Name:          "another-test-name",
					Description:   "another-test-description",
					AccessScope:   "another-access-scope",
					PermissionSet: "another-permission-set",
				},
			},
		},
		"multiple different raw configurations (role and access scope) should be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
accessScope: access-scope
permissionSet: permission-set
`),
				[]byte(`
name: test-name
description: test-description
resources:
- resource: a
  access: READ_ACCESS
- resource: b
  access: READ_WRITE_ACCESS
`),
			},
			expectedConfigurations: []Configuration{
				&Role{
					Name:          "test-name",
					Description:   "test-description",
					AccessScope:   "access-scope",
					PermissionSet: "permission-set",
				},
				&PermissionSet{
					Name:        "test-name",
					Description: "test-description",
					Resources: []ResourceWithAccess{
						{
							Resource: "a",
							Access:   Access(storage.Access_READ_ACCESS),
						},
						{
							Resource: "b",
							Access:   Access(storage.Access_READ_WRITE_ACCESS),
						},
					},
				},
			},
		},
		"invalid configuration should not be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
name: test-name
description: test-description
policy:
  name: some-poliy
  enforcement: DEPLOY_TIME
`),
			},
			fail: true,
		},
		"invalid YAML format should not be unmarshalled successfully": {
			rawConfigurations: [][]byte{
				[]byte(`
X
`),
			},
			fail: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			configurations, err := ConfigurationFromRawBytes(c.rawConfigurations...)
			if c.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.ElementsMatch(t, c.expectedConfigurations, configurations)
		})
	}
}
