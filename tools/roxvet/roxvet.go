package main

import (
	"github.com/stackrox/rox/tools/dontprintferr"
	"github.com/stackrox/rox/tools/needlessformat"
	"github.com/stackrox/rox/tools/protoclone"
	"github.com/stackrox/rox/tools/regexes"
	"github.com/stackrox/rox/tools/storedprotos/storeinterface"
	"github.com/stackrox/rox/tools/uncheckederrors"
	"github.com/stackrox/rox/tools/uncheckedifassign"
	"github.com/stackrox/rox/tools/unusedroxctlargs"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(
		dontprintferr.Analyzer,
		storeinterface.Analyzer,
		uncheckederrors.Analyzer,
		needlessformat.Analyzer,
		regexes.Analyzer,
		uncheckedifassign.Analyzer,
		protoclone.Analyzer,
		unusedroxctlargs.Analyzer,
	)
}
