package framework

import pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"

// A Check is a single piece of logic executed as part of a compliance run. It usually corresponds to one or multiple
// controls in a compliance standard.
type Check interface {
	// ID returns an ID uniquely identifying a check.
	ID() string

	// AppliesToScope checks if the check applies to the given scope. This has no effect as to how the check is executed
	// by the framework (see `Run` below), but informs how results are collected and, in particular, how missing results
	// are detected.
	AppliesToScope(scope pkgFramework.TargetKind) bool

	// Scope returns the target of the check
	Scope() pkgFramework.TargetKind

	// DataDependencies is a list of IDs for data required by a check.
	DataDependencies() []string

	// InterpretationText returns a string describing how StackRox is implementing this check.
	InterpretationText() string

	// Run is the entry point for the check logic. It is *always* invoked on a context with a 'cluster' target kind;
	// it is the responsibility of the implementation to call `RunForTarget`/`ForEachNode`/`ForEachDeployment` to cover
	// all objects at the indicated scope.
	Run(ctx ComplianceContext)
}

// CheckMetadata stores metadata associated with a check.
type CheckMetadata struct {
	ID                 string
	Scope              pkgFramework.TargetKind
	AdditionalScopes   []pkgFramework.TargetKind
	DataDependencies   []string
	InterpretationText string
	RemoteCheck        bool
}

// CheckFunc is the function realizing a compliance check. While every `Check` has a `CheckFunc` (namely `chk.Run` for
// a Check `chk`), not every `CheckFunc` corresponds to a check. Rather, a `Check` (or a `CheckFunc`) can be realized
// by invoking multiple `CheckFunc`s, e.g., one for each node/deployment in the cluster (remember that a `Check` is
// always invoked at cluster scope).
type CheckFunc func(ComplianceContext)

type checkFromFunc struct {
	metadata CheckMetadata
	checkFn  CheckFunc
}

// NewCheckFromFunc returns a new check with the given metadata from the given `CheckFunc`.
func NewCheckFromFunc(metadata CheckMetadata, checkFn CheckFunc) Check {
	return &checkFromFunc{
		metadata: metadata,
		checkFn:  checkFn,
	}
}

func (c *checkFromFunc) ID() string {
	return c.metadata.ID
}

func (c *checkFromFunc) InterpretationText() string {
	return c.metadata.InterpretationText
}

func (c *checkFromFunc) Scope() pkgFramework.TargetKind {
	return c.metadata.Scope
}

func (c *checkFromFunc) AppliesToScope(scope pkgFramework.TargetKind) bool {
	if c.metadata.Scope == scope {
		return true
	}
	for _, addlScope := range c.metadata.AdditionalScopes {
		if addlScope == scope {
			return true
		}
	}
	return false
}

func (c *checkFromFunc) DataDependencies() []string {
	return c.metadata.DataDependencies
}

func (c *checkFromFunc) Run(ctx ComplianceContext) {
	if c.metadata.RemoteCheck {
		return
	}
	c.checkFn(ctx)
}
