package declarativeconfig

// ConfigurationType specifies the type of declarative configuration.
type ConfigurationType = string

// The list of currently supported and implemented declarative configuration types.
const (
	AuthProviderConfiguration  ConfigurationType = "auth-provider"
	AccessScopeConfiguration   ConfigurationType = "access-scope"
	PermissionSetConfiguration ConfigurationType = "permission-set"
	RoleConfiguration          ConfigurationType = "role"
)

// Configuration specifies a declarative configuration.
type Configuration interface {
	Type() ConfigurationType
}
