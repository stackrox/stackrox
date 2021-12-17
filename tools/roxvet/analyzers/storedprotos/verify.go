package main

import (
	"github.com/stackrox/rox/tools/roxvet/analyzers/storedprotos/storeinterface"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(storeinterface.Analyzer)
}
