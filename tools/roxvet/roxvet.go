package main

import (
	"github.com/stackrox/rox/tools/roxvet/analyzers/dontprintferr"
	"github.com/stackrox/rox/tools/roxvet/analyzers/filepathwalk"
	"github.com/stackrox/rox/tools/roxvet/analyzers/importpackagenames"
	"github.com/stackrox/rox/tools/roxvet/analyzers/invalidoutputroxctl"
	"github.com/stackrox/rox/tools/roxvet/analyzers/needlessformat"
	"github.com/stackrox/rox/tools/roxvet/analyzers/protoclone"
	"github.com/stackrox/rox/tools/roxvet/analyzers/regexes"
	"github.com/stackrox/rox/tools/roxvet/analyzers/storeinterface"
	"github.com/stackrox/rox/tools/roxvet/analyzers/uncheckederrors"
	"github.com/stackrox/rox/tools/roxvet/analyzers/uncheckedifassign"
	"github.com/stackrox/rox/tools/roxvet/analyzers/unusedroxctlargs"
	"github.com/stackrox/rox/tools/roxvet/analyzers/validateimports"
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
		invalidoutputroxctl.Analyzer,
		filepathwalk.Analyzer,
		validateimports.Analyzer,
		importpackagenames.Analyzer,
	)
}
