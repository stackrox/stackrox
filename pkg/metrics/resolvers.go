package metrics

// Resolver represents a graphql resolver that we want to time.
//
//go:generate stringer -type=Resolver
type Resolver int

// The following is the list of graphql resolvers that we want to time.
const (
	Cluster Resolver = iota
	Compliance
	ComlianceControl
	CVEs
	Deployments
	Groups
	Images
	ImageComponents
	K8sRoles
	Namespaces
	Nodes
	Notifiers
	PermissionSets
	Policies
	Roles
	Root
	Secrets
	ServiceAccounts
	Subjects
	Tokens
	Violations
	Pods
	ContainerInstances
	ImageCVEs
	NodeCVEs
	ClusterCVEs
	NodeComponents
	ImageCVECore
)
