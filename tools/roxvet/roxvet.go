package main

import (
	"github.com/stackrox/rox/tools/dontprintferr"
	"github.com/stackrox/rox/tools/needlessformat"
	"github.com/stackrox/rox/tools/regexes"
	"github.com/stackrox/rox/tools/storedprotos/storeinterface"
	"github.com/stackrox/rox/tools/uncheckederrors"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() {
	unitchecker.Main(
		dontprintferr.Analyzer,
		storeinterface.Analyzer,
		uncheckederrors.Analyzer,
		needlessformat.Analyzer,
		regexes.Analyzer,
	)
}
