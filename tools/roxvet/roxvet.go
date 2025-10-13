package main

import (
	"github.com/stackrox/rox/tools/roxvet/analyzers/donotcompareproto"
	"github.com/stackrox/rox/tools/roxvet/analyzers/dontprintferr"
	"github.com/stackrox/rox/tools/roxvet/analyzers/filepathwalk"
	"github.com/stackrox/rox/tools/roxvet/analyzers/godoccapitalizationmismatch"
	"github.com/stackrox/rox/tools/roxvet/analyzers/gogoprotofunctions"
	"github.com/stackrox/rox/tools/roxvet/analyzers/importpackagenames"
	"github.com/stackrox/rox/tools/roxvet/analyzers/migrationreferencedschema"
	"github.com/stackrox/rox/tools/roxvet/analyzers/needlessformat"
	"github.com/stackrox/rox/tools/roxvet/analyzers/protoclone"
	"github.com/stackrox/rox/tools/roxvet/analyzers/protoptrs"
	"github.com/stackrox/rox/tools/roxvet/analyzers/regexes"
	"github.com/stackrox/rox/tools/roxvet/analyzers/sortslices"
	"github.com/stackrox/rox/tools/roxvet/analyzers/storeinterface"
	"github.com/stackrox/rox/tools/roxvet/analyzers/structuredlogs"
	"github.com/stackrox/rox/tools/roxvet/analyzers/testtags"
	"github.com/stackrox/rox/tools/roxvet/analyzers/uncheckedifassign"
	"github.com/stackrox/rox/tools/roxvet/analyzers/undeferredmutexunlocks"
	"github.com/stackrox/rox/tools/roxvet/analyzers/unmarshalreplace"
	"github.com/stackrox/rox/tools/roxvet/analyzers/unusedroxctlargs"
	"github.com/stackrox/rox/tools/roxvet/analyzers/validateimports"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(
		donotcompareproto.Analyzer,
		dontprintferr.Analyzer,
		filepathwalk.Analyzer,
		godoccapitalizationmismatch.Analyzer,
		gogoprotofunctions.Analyzer,
		importpackagenames.Analyzer,
		migrationreferencedschema.Analyzer,
		needlessformat.Analyzer,
		protoclone.Analyzer,
		protoptrs.Analyzer,
		regexes.Analyzer,
		sortslices.Analyzer,
		storeinterface.Analyzer,
		structuredlogs.Analyzer,
		testtags.Analyzer,
		uncheckedifassign.Analyzer,
		undeferredmutexunlocks.Analyzer,
		unmarshalreplace.Analyzer,
		unusedroxctlargs.Analyzer,
		validateimports.Analyzer,
	)
}
