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
