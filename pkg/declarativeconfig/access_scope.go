package declarativeconfig

import "github.com/stackrox/rox/generated/storage"

// TODO: currently it does not correspond to YAML format described in access_scope_test
type AccessScope struct {
	Name        string                           `yaml:"name,omitempty"`
	Description string                           `yaml:"description,omitempty"`
	Rules       *storage.SimpleAccessScope_Rules `yaml:"rules,omitempty"`
}
