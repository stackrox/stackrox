package resolvers

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validateAlignedMethodIndex(t testing.TB, type1 reflect.Type, type2 reflect.Type) {
	t.Helper()

	if !assert.Equalf(t, type1.NumMethod(), type2.NumMethod(),
		"Types %s and %s should have the same number of methods",
		type1.String(), type2.String(),
	) {
		dumpTypeMethods(t, type1, type1.String(), false)
		dumpTypeMethods(t, type2, type2.String(), false)
		return
	}
	for i := range type1.NumMethod() {
		m1 := type1.Method(i)
		m2 := type2.Method(i)
		assert.Equal(t, m1.Name, m2.Name)
	}
}

// dumpTypeMethods uses reflection to dump the index and names of methods for a given type.
// It accepts any resolver value and outputs method information to the test logger.
//
// Example usage:
//
//	// After creating a resolver instance in a test:
//	componentResolver, err := resolver.ImageComponent(ctx, struct{ ID *graphql.ID }{ID: &componentID})
//	require.NoError(t, err)
//	dumpTypeMethods(t, reflect.TypeOf(componentResolver), "imageComponentV2Resolver")
//
// Output includes:
//   - Index of each method (0-based)
//   - Method name
//   - Full method signature
//   - Input parameter types
//   - Output parameter types
func dumpTypeMethods(t testing.TB, typ reflect.Type, typeName string, verbose bool) {
	t.Logf("\n=== Methods of %s ===", typeName)
	t.Logf("Type: %s", typ.String())
	t.Logf("Kind: %s", typ.Kind())
	t.Logf("Number of methods: %d\n", typ.NumMethod())

	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		t.Logf("[%d] %s", i, method.Name)

		if verbose {
			// Also log the method signature
			methodType := method.Type
			t.Logf("    Signature: %s", methodType.String())

			// Log input parameters
			numIn := methodType.NumIn()
			if numIn > 0 {
				t.Logf("    Inputs (%d):", numIn)
				for j := 0; j < numIn; j++ {
					t.Logf("      [%d] %s", j, methodType.In(j).String())
				}
			}

			// Log return parameters
			numOut := methodType.NumOut()
			if numOut > 0 {
				t.Logf("    Outputs (%d):", numOut)
				for j := 0; j < numOut; j++ {
					t.Logf("      [%d] %s", j, methodType.Out(j).String())
				}
			}
			t.Logf("")
		}
	}
	t.Logf("=== End of methods for %s ===\n", typeName)
}
