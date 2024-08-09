package v4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	rpm2 "github.com/quay/claircore/rpm"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/compliance/collection/intervals"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
)

// The layer carries a hardcoded digest, as it is exclusively used for passing
// ClairCore checks, not for Scanners matching
var layerDigest = fmt.Sprintf("sha256:%s", strings.Repeat("a", 64))

type localNodeIndexer struct {
}

// NewNodeIndexer creates a new node indexer
func NewNodeIndexer() compliance.NodeIndexer {
	return &localNodeIndexer{}
}

// GetIntervals
func (l *localNodeIndexer) GetIntervals() *intervals.NodeScanIntervals {
	i := intervals.NewNodeScanIntervalFromEnv()
	return &i
}

// IndexNode indexes a live fs.FS at the container mountpoint given in the basePath.
func (l *localNodeIndexer) IndexNode(ctx context.Context) (*v4.IndexReport, error) {
	report := &claircore.IndexReport{
		Packages:      map[string]*claircore.Package{},
		Environments:  map[string][]*claircore.Environment{},
		Distributions: map[string]*claircore.Distribution{},
		Repositories:  map[string]*claircore.Repository{},
		Files:         map[string]claircore.File{},
	}

	layer, err := constructLayer(ctx, layerDigest, env.NodeScanningV4HostPath.Setting())
	if err != nil {
		return nil, err
	}
	defer func() {
		if tmpErr := layer.Close(); tmpErr != nil {
			err = tmpErr
		}
	}()

	reps, err := runRepositoryScanner(ctx, layer)
	if err != nil {
		return nil, err
	}

	pcks, err := runPackageScanner(ctx, layer)
	if err != nil {
		return nil, err
	}

	ir, err := coalesceReport(ctx, layerDigest, reps, pcks)
	if err != nil {
		return nil, err
	}

	report = controller.MergeSR(report, []*claircore.IndexReport{ir})
	report.Success = true
	report.State = controller.IndexFinished.String()

	v4Report, err := mappers.ToProtoV4IndexReport(report)
	if err != nil {
		return nil, errors.Wrap(err, "converting clair report to v4 report")
	}

	return v4Report, nil
}

func coalesceReport(ctx context.Context, layerDigest string, reps []*claircore.Repository, pcks []*claircore.Package) (*claircore.IndexReport, error) {
	ch, err := claircore.ParseDigest(layerDigest)
	if err != nil {
		return nil, errors.Wrap(err, "parsing Clair Core digest")
	}

	la := &ccindexer.LayerArtifacts{
		Hash: ch,
	}
	la.Repos = append(la.Repos, reps...)
	la.Pkgs = append(la.Pkgs, pcks...)
	artifacts := []*ccindexer.LayerArtifacts{la}
	coal := new(rhel.Coalescer)

	ir, err := coal.Coalesce(ctx, artifacts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to coalesce report")
	}
	return ir, nil
}

func runPackageScanner(ctx context.Context, layer *claircore.Layer) ([]*claircore.Package, error) {
	rpm := rpm2.Scanner{}
	pck, err := rpm.Scan(ctx, layer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to invoke RPM scanner")
	}
	pck = filterPackages(ctx, pck)
	if pck != nil {
		zlog.Info(ctx).Msgf("Num packages found: %v", len(pck))
	}
	for i, p := range pck {
		p.ID = fmt.Sprintf("%d", i)
	}
	return pck, nil
}

// As we're only interested in the effective running RPM DB,
// we filter out packages from other DBs like rpm-ostree
func filterPackages(_ context.Context, pck []*claircore.Package) []*claircore.Package {
	var filtered []*claircore.Package
	for _, pkg := range pck {
		if pkg.PackageDB == "sqlite:usr/share/rpm" {
			filtered = append(filtered, pkg)
		}
	}
	return filtered
}

func runRepositoryScanner(ctx context.Context, l *claircore.Layer) ([]*claircore.Repository, error) {
	c := http.DefaultClient
	sc := rhel.RepositoryScanner{}
	config := rhel.RepositoryScannerConfig{
		DisableAPI:         false,
		API:                "https://catalog.redhat.com/api/containers/",
		Repo2CPEMappingURL: "https://access.redhat.com/security/data/metrics/repository-to-cpe.json",
		Timeout:            10 * time.Second,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to encode configuration")
	}
	if err := sc.Configure(ctx, json.NewDecoder(&buf).Decode, c); err != nil {
		return nil, errors.Wrap(err, "failed to configure repository scanner")
	}

	reps, err := sc.Scan(ctx, l)
	if err != nil {
		return nil, errors.Wrap(err, "failed to scan repositories")
	}
	if reps != nil {
		zlog.Info(ctx).Msgf("Num repositories found: %v", len(reps))
	}
	for i, r := range reps {
		r.ID = fmt.Sprintf("%d", i)
	}
	return reps, nil
}

func constructLayer(ctx context.Context, digest string, hostPath string) (*claircore.Layer, error) {
	zlog.Info(ctx).Msgf("Realizing mount path: %s", hostPath)
	desc := &claircore.LayerDescription{
		Digest:    digest,
		URI:       hostPath,
		MediaType: "application/vnd.claircore.filesystem",
	}

	l := claircore.Layer{}
	err := l.Init(ctx, desc, nil)
	return &l, errors.Wrap(err, "failed to init layer")
}
