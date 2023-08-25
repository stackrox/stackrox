package files

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
	"github.com/stackrox/rox/tools/generate-helpers/notifier/operations"
)

// GenerateNotifierImplFile generates the implementation of the notifier: notifier_impl.go
func GenerateNotifierImplFile(variables, implementations []Code, props *operations.GeneratorProperties) error {
	f := newFile()

	f.ImportAlias(packagenames.Sync, "sync")

	f.Add(generateNewFunc(props))
	f.Line()

	// Add the variables to the struct along with the RW lock.
	structFields := append([]Code{Id("lock").Qual(packagenames.Sync, "RWMutex")}, variables...)
	f.Type().Id("notifier").Struct(structFields...)
	f.Line()

	for _, implementation := range implementations {
		f.Add(implementation)
		f.Line()
	}
	f.Line()

	return f.Save("notifier_impl.go")
}

func generateNewFunc(_ *operations.GeneratorProperties) Code {
	return Func().Id("newNotifier").Params().Parens(Op("*").Id("notifier")).Block(
		Return(Op("&").Id("notifier").Block()),
	)
}
