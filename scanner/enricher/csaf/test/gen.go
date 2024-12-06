package main

import (
	"archive/tar"
	"os"

	"github.com/klauspost/compress/zstd"
)

func main() {
	f, err := os.Create("csaf_advisories_2024-12-01.tar.zst")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	zw, err := zstd.NewWriter(f)
	if err != nil {
		panic(err)
	}
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	err = tw.AddFS(os.DirFS("/Users/rtannenb/go/src/github.com/stackrox/stackrox/scanner/enricher/csaf/testdata/"))
	if err != nil {
		panic(err)
	}
}
