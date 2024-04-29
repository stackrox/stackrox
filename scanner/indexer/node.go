package indexer

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
	ccindexer "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	rpm2 "github.com/quay/claircore/rpm"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/scanner/config"
)

// NodeIndexer represents a node indexer.
//
// It is a specialized mode of [indexer.Indexer] that takes a path and scans a live filesystem
// instead of downloading and scanning layers of a container manifest.
//
// TODO: Find out if we really need a DB for the node indexer. Likely we need a caching layer, but not a DB.
type NodeIndexer interface {
	IndexNode(ctx context.Context) (*claircore.IndexReport, error)
	Close(ctx context.Context) error
}

type localNodeIndexer struct {
	client http.Client
}

// NewNodeIndexer creates a new node indexer
func NewNodeIndexer(ctx context.Context, cfg config.NodeIndexerConfig) (NodeIndexer, error) {
	// ctx = zlog.ContextWithValues(ctx, "component", "scanner/backend/indexer.NewNodeIndexer")

	// Note: http.DefaultTransport has already been modified to handle configured proxies.
	// See scanner/cmd/scanner/main.go.
	// t, err := httputil.TransportMux(http.DefaultTransport, httputil.WithDenyStackRoxServices(!cfg.StackRoxServices))
	// if err != nil {
	//	return nil, fmt.Errorf("creating HTTP transport: %w", err)
	//}
	// client := &http.Client{
	//	Transport: t,
	//}

	return &localNodeIndexer{}, nil
}

// IndexNode indexes a live fs.FS at the container mountpoint given in the basePath.
func (l *localNodeIndexer) IndexNode(ctx context.Context) (*claircore.IndexReport, error) {
	report := &claircore.IndexReport{
		Packages:      map[string]*claircore.Package{},
		Environments:  map[string][]*claircore.Environment{},
		Distributions: map[string]*claircore.Distribution{},
		Repositories:  map[string]*claircore.Repository{},
		Files:         map[string]claircore.File{},
	}

	h := getRandomSHA256()
	ch, err := claircore.ParseDigest(`sha256:` + h)
	if err != nil {
		return nil, err
	}
	report.Hash = ch

	// SA1029 FIXME: Find a better way to pass the manifest ID through the stack
	//nolint:staticcheck
	ctx = context.WithValue(ctx, "manifest_id", h)

	layer, err := constructLayer(ctx)
	if err != nil {
		return nil, err
	}

	reps, err := runRepositoryScanner(ctx, layer)
	if err != nil {
		return nil, err
	}

	// package scanner
	rpm := rpm2.Scanner{}
	pck, err := rpm.Scan(ctx, layer)
	if err != nil {
		return nil, err
	}
	if pck != nil {
		zlog.Info(ctx).Msgf("Num packages found: %v", len(pck))
	}
	for i, p := range pck {
		p.ID = fmt.Sprintf("%d", i)
	}

	// coalesce
	la := &ccindexer.LayerArtifacts{
		Hash: claircore.MustParseDigest(`sha256:` + h),
	}
	la.Repos = append(la.Repos, reps...)
	la.Pkgs = append(la.Pkgs, pck...)
	artifacts := []*ccindexer.LayerArtifacts{la}
	coal := new(rhel.Coalescer)

	ir, err := coal.Coalesce(ctx, artifacts)
	if err != nil {
		panic(err)
	}
	report = controller.MergeSR(report, []*claircore.IndexReport{ir})
	report.Success = true
	report.State = controller.IndexFinished.String()

	return report, nil
}

// Close closes the NodeIndexer.
func (l *localNodeIndexer) Close(ctx context.Context) error {
	return nil
}

// Ready check function.
func (l *localNodeIndexer) Ready(_ context.Context) error {
	return nil
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
		return nil, err
	}
	if err := sc.Configure(ctx, json.NewDecoder(&buf).Decode, c); err != nil {
		return nil, err
	}

	reps, err := sc.Scan(ctx, l)
	if err != nil {
		return nil, err
	}
	if reps != nil {
		zlog.Info(ctx).Msgf("Num repositories found: %v", len(reps))
	}
	for i, r := range reps {
		r.ID = fmt.Sprintf("%d", i)
	}
	return reps, nil
}

func getRandomSHA256() string {
	data := make([]byte, 10)
	_, err := rand.Read(data)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func constructLayer(ctx context.Context) (*claircore.Layer, error) {
	hostPath := env.NodeScanningV4HostPath.Setting()
	zlog.Info(ctx).Msgf("Realizing mount path: %s", hostPath)
	nodeFS := os.DirFS(hostPath)
	l := claircore.Layer{}
	err := l.InitROFS(ctx, nodeFS)
	if err != nil {
		return nil, err
	}
	return &l, nil
}
