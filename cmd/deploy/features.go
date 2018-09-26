package main

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/features"
)

var (
	flagKeys = make(map[string]features.FeatureFlag)
)

func init() {
	for _, flag := range features.Flags {
		flagKeys[flag.EnvVar()] = flag
	}
}

type featureValue struct {
	featureSlice *[]features.FeatureFlag
}

func (v *featureValue) String() string {
	out := make([]string, len(*v.featureSlice))
	for i, f := range *v.featureSlice {
		out[i] = f.EnvVar()
	}
	return strings.Join(out, ",")
}

func (v *featureValue) Set(input string) error {
	for _, flagKey := range strings.Split(input, ",") {
		flag, ok := flagKeys[flagKey]
		if !ok {
			return fmt.Errorf("flag not found: %s", flagKey)
		}
		*v.featureSlice = append(*v.featureSlice, flag)
	}
	return nil
}

func (v *featureValue) Type() string {
	return "flagSlice"
}
