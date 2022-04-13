package printer

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/utils"
)

func registerFunc(key string, f Func) {
	if _, ok := knownFuncs[key]; ok {
		utils.CrashOnError(errors.Errorf("duplicate key: %s", key))
	}
	knownFuncs[key] = f
}

var (
	knownFuncs = make(map[string]Func)
)

// GetFuncs gets the functions with the corresponding keys.
func GetFuncs(keys set.StringSet) []Func {
	out := make([]Func, 0, len(keys))
	for k := range keys {
		out = append(out, knownFuncs[k])
	}
	return out
}
