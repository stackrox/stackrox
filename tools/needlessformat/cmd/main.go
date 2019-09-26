package main

import (
	"github.com/stackrox/rox/tools/needlessformat"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(needlessformat.Analyzer)
}
