package services

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

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	indexer2 "github.com/quay/claircore/indexer"
	"github.com/quay/claircore/indexer/controller"
	"github.com/quay/claircore/rhel"
	rpm2 "github.com/quay/claircore/rpm"
	"github.com/quay/zlog"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/mappers"
	"google.golang.org/grpc"
)

var (
	_ v4.NodeIndexerServer = (*nodeIndexerService)(nil)
)

type nodeIndexerService struct {
	v4.UnimplementedNodeIndexerServer
	nodeIndexer indexer.NodeIndexer
}

// NewNodeIndexerService returns a new NodeIndexerService
func NewNodeIndexerService(indexer indexer.NodeIndexer) *nodeIndexerService {
	return &nodeIndexerService{nodeIndexer: indexer}
}

// CreateNodeIndexReport is the endpoint to create a new report for the node it runs on.
func (s *nodeIndexerService) CreateNodeIndexReport(ctx context.Context, _ *v4.CreateNodeIndexReportRequest) (*v4.IndexReport, error) {
	// clairReport, err := s.nodeIndexer.IndexNode(ctx)
	clairReport, err := createNodeIndexReport(ctx)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("nodeIndexer.IndexNode failed")
		return nil, err
	}

	if !clairReport.Success {
		return nil, fmt.Errorf("internal error: create node index report failed in state %q: %s", clairReport.State, clairReport.Err)
	}

	indexReport, err := mappers.ToProtoV4IndexReport(clairReport)
	if err != nil {
		zlog.Error(ctx).Err(err).Msg("internal error: converting node index to v4.IndexReport")
		return nil, err
	}

	return indexReport, nil
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *nodeIndexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// If this a dev build, allow anonymous traffic for testing purposes.
	if !buildinfo.ReleaseBuild {
		auth := allow.Anonymous()
		return ctx, auth.Authorized(ctx, fullMethodName)
	}

	// FIXME: Set up auth for prod builds
	return ctx, errors.New("Not implemented / unauthorized")
}

// RegisterServiceServer .
func (s *nodeIndexerService) RegisterServiceServer(server *grpc.Server) {
	v4.RegisterNodeIndexerServer(server, s)
}

// RegisterServiceHandler .
func (s *nodeIndexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the indexer.
	return nil
}

func createNodeIndexReport(ctx context.Context) (*claircore.IndexReport, error) {
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

	ctx = context.WithValue(ctx, "manifest_id", h)

	l, err := constructLayer(ctx)
	if err != nil {
		return nil, err
	}

	reps, err := runRepositoryScanner(ctx, l)
	if err != nil {
		return nil, err
	}

	// package scanner
	rpm := rpm2.Scanner{}
	pck, err := rpm.Scan(ctx, l)
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
	la := &indexer2.LayerArtifacts{
		Hash: claircore.MustParseDigest(`sha256:` + h),
	}
	la.Repos = append(la.Repos, reps...)
	la.Pkgs = append(la.Pkgs, pck...)
	artifacts := []*indexer2.LayerArtifacts{la}
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
	rand.Read(data)
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
