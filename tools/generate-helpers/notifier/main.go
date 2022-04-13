package main

import (
	"fmt"
	"os"
	"sort"

	. "github.com/dave/jennifer/jen"
	"github.com/spf13/cobra"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stackrox/stackrox/pkg/utils"
	"github.com/stackrox/stackrox/tools/generate-helpers/notifier/files"
	"github.com/stackrox/stackrox/tools/generate-helpers/notifier/operations"
)

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	props := operations.GeneratorProperties{}
	c.Flags().StringVar(&props.Pkg, "package", "github.com/stackrox/stackrox/generated/storage", "the package of the object generating notifications")

	c.Flags().StringVar(&props.Object, "object", "", "the (Go) name of the object sent in the notification")
	utils.Must(c.MarkFlagRequired("object"))

	methods := c.Flags().StringSlice("methods", nil, fmt.Sprintf("the methods to generate (supported - %s)", operations.RenderSupportedMethods()))
	utils.Must(c.MarkFlagRequired("methods"))

	c.RunE = func(*cobra.Command, []string) error {
		if err := checkSupported(*methods); err != nil {
			return err
		}
		return generate(&props, *methods)
	}

	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func checkSupported(methods []string) error {
	for _, method := range methods {
		if !operations.IsSupported(method) {
			return fmt.Errorf("method %q is unsupported", method)
		}
	}
	return nil
}

func generate(props *operations.GeneratorProperties, methods []string) error {
	methodsSet := set.NewStringSet(methods...)
	signatures, variables, implementations := generateFunctions(props, methodsSet.AsSlice())

	if err := files.GenerateSignatureFile(signatures); err != nil {
		return err
	}
	if err := files.GenerateNotifierImplFile(variables, implementations, props); err != nil {
		return err
	}
	return nil
}

func generateFunctions(props *operations.GeneratorProperties, methods []string) (signatures []Code, variables []Code, implementations []Code) {
	// Generate code in a deterministic order so the style checker doesn't complain about stale generated code
	sort.Strings(methods)
	for _, method := range methods {
		newSignatures, newVariables, newImplementations := operations.GenerateSignaturesAndImplementations(method, props)
		signatures = append(signatures, newSignatures...)
		implementations = append(implementations, newImplementations...)
		variables = append(variables, newVariables...)
	}
	return
}
