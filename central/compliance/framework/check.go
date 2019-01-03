package framework

// A Check is a single piece of logic executed as part of a compliance run. It usually corresponds to one or multiple
// controls in a compliance standard.
type Check interface {
	// ID returns an ID uniquely identifying a check.
	ID() string
	// Scope is the scope at which the check operates. This has no effect as to how the check is executed by the
	// framework (see `Run` below), but informs how results are collected and, in particular, how missing results are
	// detected.
	Scope() TargetKind
	// DataDependencies is a list of IDs for data required by a check.
	DataDependencies() []string

	// Run is the entry point for the check logic. It is *always* invoked on a context with a 'cluster' target kind;
	// it is the responsibility of the implementation to call `RunForTarget`/`ForEachNode`/`ForEachDeployment` to cover
	// all objects at the indicated scope.
	Run(ctx ComplianceContext)
}

// CheckFunc is the function realizing a compliance check. While every `Check` has a `CheckFunc` (namely `chk.Run` for
// a Check `chk`), not every `CheckFunc` corresponds to a check. Rather, a `Check` (or a `CheckFunc`) can be realized
// by invoking multiple `CheckFunc`s, e.g., one for each node/deployment in the cluster (remember that a `Check` is
// always invoked at cluster scope).
type CheckFunc func(ComplianceContext)

type checkFromFunc struct {
	id               string
	scope            TargetKind
	dataDependencies []string
	checkFn          CheckFunc
}

// NewCheckFromFunc returns a new check with the given metadata from the given `CheckFunc`.
func NewCheckFromFunc(id string, scope TargetKind, dataDependencies []string, checkFn CheckFunc) Check {
	return &checkFromFunc{
		id:               id,
		scope:            scope,
		dataDependencies: dataDependencies,
		checkFn:          checkFn,
	}
}

func (c *checkFromFunc) ID() string {
	return c.id
}

func (c *checkFromFunc) Scope() TargetKind {
	return c.scope
}

func (c *checkFromFunc) DataDependencies() []string {
	return c.dataDependencies
}

func (c *checkFromFunc) Run(ctx ComplianceContext) {
	c.checkFn(ctx)
}
