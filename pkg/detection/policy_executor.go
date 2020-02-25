package detection

// PolicyExecutor runs a compiled policy and returns an error.
type PolicyExecutor interface {
	Execute(CompiledPolicy) error
}

// FunctionAsExecutor wraps an input function as the Executor function of a PolicyExecutor.
func FunctionAsExecutor(f func(CompiledPolicy) error) PolicyExecutor {
	return &functionWrappingExecutor{
		f: f,
	}
}

type functionWrappingExecutor struct {
	f func(CompiledPolicy) error
}

func (fwe *functionWrappingExecutor) Execute(cp CompiledPolicy) error {
	return fwe.f(cp)
}
