package files

import (
	. "github.com/dave/jennifer/jen"
)

// GenerateSignatureFile generates the signature of the Notifier interface and it's constructor: notifier.go
func GenerateSignatureFile(signatures []Code) error {
	f := newFile()
	f.Type().Id("Notifier").Interface(signatures...)
	f.Func().Id("New").Params().Id("Notifier").Block(
		Return(Id("newNotifier").Call()),
	)
	return f.Save("notifier.go")
}
