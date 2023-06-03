package declarativeconfig

// Role is representation of storage.Role that supports transformation from YAML.
type Role struct {
	Name          string `yaml:"name,omitempty"`
	Description   string `yaml:"description,omitempty"`
	AccessScope   string `yaml:"accessScope,omitempty"`
	PermissionSet string `yaml:"permissionSet,omitempty"`
}

// ConfigurationType returns the RoleConfiguration type.
func (r *Role) ConfigurationType() ConfigurationType {
	return RoleConfiguration
}
