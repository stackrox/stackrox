package main

import (
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/dontprintferr"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/filepathwalk"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/godoccapitalizationmismatch"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/importpackagenames"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/needlessformat"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/protoclone"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/regexes"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/storeinterface"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/uncheckederrors"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/uncheckedifassign"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/unusedroxctlargs"
	"github.com/stackrox/stackrox/tools/roxvet/analyzers/validateimports"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(
		godoccapitalizationmismatch.Analyzer,
		dontprintferr.Analyzer,
		storeinterface.Analyzer,
		uncheckederrors.Analyzer,
		needlessformat.Analyzer,
		regexes.Analyzer,
		uncheckedifassign.Analyzer,
		protoclone.Analyzer,
		unusedroxctlargs.Analyzer,
		filepathwalk.Analyzer,
		validateimports.Analyzer,
		importpackagenames.Analyzer,
	)
}
