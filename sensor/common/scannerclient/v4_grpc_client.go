package scannerclient

import (
	"context"
	"sort"
	"strings"

	"github.com/pkg/errors"
	scannerV4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/registries/types"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var _ Client = (*V4GRPCClient)(nil)

// V4GRPCClient represents a client implementation using the v4 gRPC protocol.
type V4GRPCClient struct {
	indexerClient scannerV4.IndexerClient
	matcherClient scannerV4.MatcherClient
	conn          *grpc.ClientConn
}

func (v V4GRPCClient) Dial(endpoint string) (Client, error) {
	if endpoint == "" {
		return nil, errors.New("Invalid Scanner endpoint (empty)")
	}

	endpoint = strings.TrimPrefix(endpoint, "https://")
	if strings.Contains(endpoint, "://") {
		return nil, errors.Errorf("ScannerV4 endpoint has unsupported scheme: %s", endpoint)
	}

	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize ScannerV4 TLS config")
	}

	// This is non-blocking. If we ever want this to block,
	// then add the grpc.WithBlock() DialOption.
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	if err != nil {
		return nil, errors.Wrap(err, "failed to dial Scanner")
	}

	log.Infof("Dialing ScannerV4 at %s", endpoint)

	return &V4GRPCClient{
		indexerClient: scannerV4.NewIndexerClient(conn),
		matcherClient: scannerV4.NewMatcherClient(conn),
		conn:          conn,
	}, nil
}

func (v V4GRPCClient) GetImageAnalysis(ctx context.Context, image *storage.Image, cfg *types.Config) (*scannerV1.GetImageComponentsResponse, error) {
	name := image.GetName().GetFullName()
	indexReport, err := v.indexerClient.CreateIndexReport(ctx, &scannerV4.CreateIndexReportRequest{
		HashId:               "",
		ResourceLocator:      nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}, grpc.WaitForReady(true))

	if err != nil {
		log.Debugf("Unable to get image components from local Scanner for image %s: %v", name, err)
		return nil, errors.Wrap(err, "getting image components from scanner")
	}

	log.Debugf("Received image indexing report from local Scanner for image %s", name)

	//convert indexReport to scannerV4.CreateIndexReportResponse
	resp, err := convertIndexReportToV1GetImageComponentsResponse(*indexReport, image)
	//return resp or return indexReport directly?
	if err != nil {
		log.Debugf("Failed to convert indexer report to image components from local Scanner for image %s: %v", name, err)
		return nil, errors.Wrap(err, "converting indexer report from scanner")
	}

	return resp, nil
}

func convertIndexReportToV1GetImageComponentsResponse(indexReport scannerV4.IndexReport, image *storage.Image) (*scannerV1.GetImageComponentsResponse, error) {
	res := &scannerV1.GetImageComponentsResponse{}
	if indexReport.Success {
		res.Status = scannerV1.ScanStatus_SUCCEEDED
		res.ScannerVersion = image.GetScan().ScannerVersion
		// TODO: Convert indexReport package information to scannerV1.GetImageComponentsResponse components
		res.Components.Namespace = getNamespace(indexReport)
	} else {
		if len(indexReport.Err) > 0 {
			return nil, errors.New(indexReport.Err)
		}
		return nil, errors.New("Failed to fetch index report")
	}
	return res, nil
}

func getNamespace(indexReport scannerV4.IndexReport) string {
	distributions := indexReport.GetDistributions()
	sort.Slice(distributions, func(i, j int) bool {
		return distributions[i].Name < distributions[j].Name
	})
	return distributions[0].Name
}

func (v V4GRPCClient) Close() error {
	return v.conn.Close()
}
