package main

import (
	"github.com/stackrox/rox/tools/dontprintferr"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(dontprintferr.Analyzer)
}
