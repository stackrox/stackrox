package main

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/stackrox/rox/compliance/virtualmachines/testdata"
)

type fixtureSpec struct {
	filename string
	opts     testdata.Options
}

func main() {
	outDir := flag.String("out-dir", ".", "Directory where fixture files will be written")
	flag.Parse()

	specs := []fixtureSpec{
		{
			filename: "indexreport_small.pb",
			opts: testdata.Options{
				VsockCID:        101,
				NumPackages:     500,
				NumRepositories: 50,
			},
		},
		{
			filename: "indexreport_avg.pb",
			opts: testdata.Options{
				VsockCID:        202,
				NumPackages:     700,
				NumRepositories: 70,
			},
		},
		{
			filename: "indexreport_large.pb",
			opts: testdata.Options{
				VsockCID:        303,
				NumPackages:     1500,
				NumRepositories: 150,
			},
		},
	}

	for _, spec := range specs {
		path := filepath.Join(*outDir, spec.filename)
		if err := testdata.WriteFixture(path, spec.opts); err != nil {
			panic(fmt.Sprintf("writing fixture %s: %v", spec.filename, err))
		}
		fmt.Printf("wrote %s\n", path)
	}
}
