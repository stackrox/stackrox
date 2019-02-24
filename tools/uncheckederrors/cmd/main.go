package main

import (
	"github.com/stackrox/rox/tools/uncheckederrors"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(uncheckederrors.Analyzer)
}
