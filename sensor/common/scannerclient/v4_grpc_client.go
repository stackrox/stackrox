package scannerclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	scannerV4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/images/utils"
	imageutils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/registries/types"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var _ Client = (*V4GRPCClient)(nil)
var supportedLanguageComponents = map[string]scannerV1.SourceType{
	"go":       scannerV1.SourceType_UNSET_SOURCE_TYPE, //TODO add a new type for Go
	"maven":    scannerV1.SourceType_JAVA,
	"pypi":     scannerV1.SourceType_PYTHON,
	"rubygems": scannerV1.SourceType_GEM,
}

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

func (v V4GRPCClient) GetImageAnalysis(ctx context.Context, image *storage.Image, cfg *types.Config) (*scannerV1.GetImageComponentsResponse, *scannerV4.IndexReport, error) {
	name := image.GetName().GetFullName()
	hid, err := claircore.ParseDigest(imageutils.GetSHA(image))
	if err != nil {
		log.Debugf("Unable to parse claircore image digest from local Scanner for image %s: %v", name, err)
		return nil, nil, errors.Wrap(err, "getting image components from scanner")
	}
	scheme := "https"
	if cfg.Insecure {
		scheme = "http"
	}
	imgUrl := fmt.Sprintf("%s://%s", scheme, utils.GetFullyQualifiedFullName(image))
	indexReport, err := v.indexerClient.CreateIndexReport(ctx, &scannerV4.CreateIndexReportRequest{
		HashId: hid.String(),
		ResourceLocator: &scannerV4.CreateIndexReportRequest_ContainerImage{ContainerImage: &scannerV4.ContainerImageLocator{
			Url:      imgUrl,
			Username: cfg.Username,
			Password: cfg.Password,
		}},
	}, grpc.WaitForReady(true))

	if err != nil {
		log.Debugf("Unable to get image components from local Scanner for image %s: %v", name, err)
		return nil, nil, errors.Wrap(err, "Fail to get image components from scanner")
	}

	log.Infof("Received image indexer report from local scanner for image %s", name)

	// Convert indexReport to scannerV1.GetImageComponentsResponse
	if indexReport == nil {
		err = errors.New("Failed to get scanner v4 indexer report")
		log.Debugf("Scanner v4 indexer report status is nil for image %s", name)
		return nil, nil, err
	}
	res, err := convertIndexReportToV1GetImageComponentsResp(indexReport, image)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Fail to convert scanner v4 indexer report")
	}
	return res, indexReport, nil
}

func convertIndexReportToV1GetImageComponentsResp(report *scannerV4.IndexReport, image *storage.Image) (*scannerV1.GetImageComponentsResponse, error) {
	res := &scannerV1.GetImageComponentsResponse{}
	if !report.Success {
		if len(report.Err) > 0 {
			return nil, errors.New(report.Err)
		}
		return nil, errors.New("Failed to fetch index report")
	}
	res.Status = scannerV1.ScanStatus_SUCCEEDED
	res.ScannerVersion = image.GetScan().ScannerVersion
	// Convert report package information to scannerV1.GetImageComponentsResponse components
	ns, isRhelComponent := getNamespace(report)
	res.Components.Namespace = ns
	repoMap := repoSliceToMap(report.Repositories)
	for _, pkg := range report.Packages {
		envMap := report.Environments
		if _, exists := envMap[pkg.Id]; !exists {
			continue
		}
		envList := envMap[pkg.Id].GetEnvironments()
		isLanguageComponent := false
		// Check if it's a language component
		for _, env := range envList {
			for _, repoId := range env.RepositoryIds {
				repoName := repoMap[repoId].Name
				if _, exists := supportedLanguageComponents[repoName]; exists {
					// If language component exists
					lngComponent := &scannerV1.LanguageComponent{
						Type:     supportedLanguageComponents[repoName],
						Name:     pkg.Name,
						Version:  pkg.Version,
						Location: env.PackageDb,    // Where the package is discovered
						AddedBy:  env.IntroducedIn, // Which layer the package is discovered
					}
					res.Components.LanguageComponents = append(res.Components.LanguageComponents, lngComponent)
					isLanguageComponent = true
					break
				}
			}
			//for current pkg in current env, it must be either language component or not ?
			if !isLanguageComponent {
				if isRhelComponent { // Process as RHEL component
					rhelComponent := &scannerV1.RHELComponent{
						Namespace: ns,
						Name:      pkg.Name,
						AddedBy:   env.IntroducedIn,
						Version:   pkg.Version,
						Arch:      report.Distributions[0].Arch, // same as getNamespace(), assume there's only one distribution
						Module:    "",                           // TODO need to figure out module
						Cpes:      nil,                          // TODO need to get CPEs from repositories
					}
					res.Components.RhelComponents = append(res.Components.RhelComponents, rhelComponent)
				} else {
					osComponent := &scannerV1.OSComponent{
						Namespace:   ns,
						AddedBy:     env.IntroducedIn,
						Version:     pkg.Version,
						Name:        pkg.Name,
						Executables: nil, // TODO: Find executables in OS component
					}
					res.Components.OsComponents = append(res.Components.OsComponents, osComponent)
				}
			}
		}

	}
	return res, nil
}

func getNamespace(report *scannerV4.IndexReport) (string, bool) {
	if len(report.Distributions) == 1 {
		dist := report.Distributions[0]
		os := dist.Name + ":" + dist.Version
		return os, dist.Did == "rhel"
	}
	return "unknown", false
}

// repoSliceToMap is to convert repository slice to map
func repoSliceToMap(rs []*scannerV4.Repository) map[string]scannerV4.Repository {
	repoMap := make(map[string]scannerV4.Repository)
	if rs != nil {
		for _, repo := range rs {
			repoMap[repo.Id] = *repo
		}
	}
	return repoMap
}

func (v V4GRPCClient) Close() error {
	return v.conn.Close()
}
