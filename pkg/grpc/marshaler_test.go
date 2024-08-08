package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MarshalerTest struct {
	suite.Suite
}

func (a *MarshalerTest) SetupTest() {
	// In order to start the gRPC server, we need to have the MTLS environment variables
	// pointing to some valid certificate/key pair. In this case we are using the ones
	// created for local-sensor, which are dummy self-signed certificates.
	a.T().Setenv("ROX_MTLS_CERT_FILE", "../../tools/local-sensor/certs/cert.pem")
	a.T().Setenv("ROX_MTLS_KEY_FILE", "../../tools/local-sensor/certs/key.pem")
	a.T().Setenv("ROX_MTLS_CA_FILE", "../../tools/local-sensor/certs/caCert.pem")
	a.T().Setenv("ROX_MTLS_CA_KEY_FILE", "../../tools/local-sensor/certs/caKey.pem")
}
func Test_MarshallerTest(t *testing.T) {
	suite.Run(t, new(MarshalerTest))
}

var _ suite.SetupTestSuite = &MarshalerTest{}

// Testing server error response from gRPC Gateway.
type supressCveServiceTestErrorImpl struct {
	v1.UnimplementedNodeCVEServiceServer
}

func (s *supressCveServiceTestErrorImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNodeCVEServiceServer(grpcServer, s)
}

func (s *supressCveServiceTestErrorImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, grpcServer *grpc.ClientConn) error {
	return v1.RegisterNodeCVEServiceHandler(ctx, mux, grpcServer)
}

func (s *supressCveServiceTestErrorImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

func (s *supressCveServiceTestErrorImpl) SuppressCVEs(_ context.Context, req *v1.SuppressCVERequest) (*v1.Empty, error) {
	return nil, status.Error(codes.Canceled, req.String())
}

func (a *MarshalerTest) TestDurationParsing() {
	url := "https://localhost:8080/v1/nodecves/suppress"

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	api := NewAPI(defaultConf())
	grpcServiceHandler := &supressCveServiceTestErrorImpl{}
	api.Register(grpcServiceHandler)
	a.Assert().NoError(api.Start().Wait())
	a.T().Cleanup(func() { api.Stop() })

	b := strings.NewReader(`{"cves": ["ABC", "XYZ"],  "duration": "24h"}`)

	req, err := http.NewRequest(http.MethodPatch, url, b)
	a.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	a.NoError(err)

	body, err := io.ReadAll(resp.Body)
	a.Require().NoError(err)

	str := `cves:"ABC"  cves:"XYZ"  duration:{seconds:86400}`
	expected := fmt.Sprintf(`{"code":1, "details": [], "error":%q, "message":%q}`, str, str)
	a.Assert().JSONEq(expected, string(body))
}
