package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	rpm2 "github.com/quay/claircore/rpm"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/mappers"
)

func main() {
	report := &claircore.IndexReport{
		Packages:      map[string]*claircore.Package{},
		Environments:  map[string][]*claircore.Environment{},
		Distributions: map[string]*claircore.Distribution{},
		Repositories:  map[string]*claircore.Repository{},
		Files:         map[string]claircore.File{},
	}

	h := getRandomSHA256()
	c := http.DefaultClient
	ctx := context.TODO()
	ctx = context.WithValue(ctx, "manifest_id", h)

	// construct a layer
	zlog.Info(ctx).Msgf("Realizing mount path: %s", "/tmp/rhcos")
	nodeFS := os.DirFS("/tmp/rhcos")
	l := claircore.Layer{}
	err := l.InitROFS(ctx, nodeFS)
	if err != nil {
		panic(err)
	}

	// repository scanner
	sc := rhel.RepositoryScanner{}
	config := rhel.RepositoryScannerConfig{
		DisableAPI:         false,
		API:                "https://catalog.redhat.com/api/containers/",
		Repo2CPEMappingURL: "https://access.redhat.com/security/data/metrics/repository-to-cpe.json",
		Timeout:            10 * time.Second,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&config); err != nil {
		panic(err)
	}
	if err := sc.Configure(ctx, json.NewDecoder(&buf).Decode, c); err != nil {
		panic(err)
	}

	reps, err := sc.Scan(ctx, &l)
	if err != nil {
		panic(err)
	}
	if reps != nil {
		zlog.Info(ctx).Msgf("Num repositories found: %v", len(reps))
	}
	for i, r := range reps {
		r.ID = fmt.Sprintf("%d", i)
	}

	// package scanner
	rpm := rpm2.Scanner{}
	pck, err := rpm.Scan(ctx, &l)
	if err != nil {
		panic(err)
	}
	if pck != nil {
		zlog.Info(ctx).Msgf("Num packages found: %v", len(pck))
	}
	for i, p := range pck {
		p.ID = fmt.Sprintf("%d", i)
	}

	// coalesce
	la := &indexer.LayerArtifacts{
		Hash: claircore.MustParseDigest(`sha256:` + h),
	}
	la.Repos = append(la.Repos, reps...)
	la.Pkgs = append(la.Pkgs, pck...)
	artifacts := []*indexer.LayerArtifacts{la}
	coal := new(rhel.Coalescer)

	ir, err := coal.Coalesce(ctx, artifacts)
	if err != nil {
		panic(err)
	}
	report = controller.MergeSR(report, []*claircore.IndexReport{ir})
	report.Success = true
	report.State = controller.IndexFinished.String()

	// convert and marshal to json
	r, err := mappers.ToProtoV4IndexReport(report)
	if err != nil {
		panic(err)
	}
	reportJSON, err := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(reportJSON))

}

func getRandomSHA256() string {
	data := make([]byte, 10)
	rand.Read(data)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
