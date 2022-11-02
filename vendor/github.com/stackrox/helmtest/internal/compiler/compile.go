package compiler

import "github.com/itchyny/gojq"

// Compile compiles a given query, applying the given options. This function mainly differs from `gojq.Compile`
// in that it injects builtin functions.
func Compile(query *gojq.Query, compilerOpts ...gojq.CompilerOption) (*gojq.Code, error) {
	var allOpts []gojq.CompilerOption
	if len(compilerOpts) == 0 {
		allOpts = builtinOpts
	} else {
		allOpts = make([]gojq.CompilerOption, 0, len(compilerOpts)+len(builtinOpts))
		allOpts = append(allOpts, builtinOpts...)
		allOpts = append(allOpts, compilerOpts...)
	}
	return gojq.Compile(query, allOpts...)
}
