package declarativeconfig

import (
	"maps"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"go.yaml.in/yaml/v3"
)

// PermissionSet is representation of storage.PermissionSet that supports transformation from YAML.
type PermissionSet struct {
	Name             string               `yaml:"name,omitempty"`
	Description      string               `yaml:"description,omitempty"`
	Resources        []ResourceWithAccess `yaml:"resources,omitempty"`
	AccessConstraint AccessConstraint     `yaml:"accessConstraint,omitempty"`
	ScopeConstraint  ScopeConstraint      `yaml:"scopeConstraint,omitempty"`
}

// AccessConstraint is the YAML representation of storage.AccessConstraint.
type AccessConstraint storage.AccessConstraint

// MarshalYAML transforms AccessConstraint to its protobuf name.
func (a AccessConstraint) MarshalYAML() (interface{}, error) {
	return storage.AccessConstraint(a).String(), nil
}

// UnmarshalYAML parses an AccessConstraint from its protobuf name.
func (a *AccessConstraint) UnmarshalYAML(value *yaml.Node) error {
	var name string
	if err := value.Decode(&name); err != nil {
		return err
	}
	constraint, ok := storage.AccessConstraint_value[name]
	if !ok {
		return errox.InvalidArgs.Newf("access constraint %q is invalid", name)
	}
	*a = AccessConstraint(constraint)
	return nil
}

// ScopeConstraint is the YAML representation of storage.ScopeConstraint.
type ScopeConstraint storage.ScopeConstraint

// MarshalYAML transforms ScopeConstraint to its protobuf name.
func (s ScopeConstraint) MarshalYAML() (interface{}, error) {
	return storage.ScopeConstraint(s).String(), nil
}

// UnmarshalYAML parses a ScopeConstraint from its protobuf name.
func (s *ScopeConstraint) UnmarshalYAML(value *yaml.Node) error {
	var name string
	if err := value.Decode(&name); err != nil {
		return err
	}
	constraint, ok := storage.ScopeConstraint_value[name]
	if !ok {
		return errox.InvalidArgs.Newf("scope constraint %q is invalid", name)
	}
	*s = ScopeConstraint(constraint)
	return nil
}

// ConfigurationType returns the PermissionSetConfiguration type.
func (p *PermissionSet) ConfigurationType() ConfigurationType {
	return PermissionSetConfiguration
}

// Access is representation of storage.Access that supports transformation from YAML.
type Access storage.Access

// ResourceWithAccess unites resource name and corresponding access level.
type ResourceWithAccess struct {
	Resource string `yaml:"resource,omitempty"`
	Access   Access `yaml:"access,omitempty"`
}

// MarshalYAML transforms Access to YAML format.
func (a Access) MarshalYAML() (interface{}, error) {
	protoAccess := storage.Access(a)
	return protoAccess.String(), nil
}

// UnmarshalYAML makes transformation from YAML to Access.
func (a *Access) UnmarshalYAML(value *yaml.Node) error {
	var v string
	if err := value.Decode(&v); err != nil {
		return err
	}
	i, ok := storage.Access_value[v]
	if !ok {
		return errox.InvalidArgs.Newf("access %s is invalid, valid values are: [%s]", v, strings.Join(
			slices.Collect(maps.Keys(storage.Access_value)), ","))
	}
	*a = Access(i)
	return nil
}
