package certdistribution

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/services"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/square/go-jose.v2/jwt"
	v1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	authenticationV1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
)

const (
	// cacheDir is the directory in which certificates to be distributed are stored.
	cacheDir = `/var/cache/stackrox/.certificates`

	maxQueryRate rate.Limit = 1.0

	maxBurstRequests = 10
)

var (
	authorizer = allow.Anonymous() // allow anonymous access because we verify tokens directly with the API server
)

type service struct {
	namespace string

	k8sAuthnClient authenticationV1.AuthenticationV1Interface

	rateLimiter *rate.Limiter
}

func newService(k8sClient kubernetes.Interface, namespace string) *service {
	return &service{
		namespace:      namespace,
		k8sAuthnClient: k8sClient.AuthenticationV1(),
		rateLimiter:    rate.NewLimiter(maxQueryRate, maxBurstRequests),
	}
}

func (s *service) RegisterServiceServer(grpcSrv *grpc.Server) {
	sensor.RegisterCertDistributionServiceServer(grpcSrv, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, cc *grpc.ClientConn) error {
	return nil
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) verifyToken(ctx context.Context, token string, expectedSubject string) error {
	parsedToken, err := jwt.ParseSigned(token)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid JWT: %s", err)
	}

	var claims map[string]interface{}
	if err := parsedToken.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return status.Errorf(codes.InvalidArgument, "unparseable claims in token: %s", err)
	}

	if sub, ok := claims["sub"].(string); !ok {
		return status.Error(codes.InvalidArgument, "non-string subject claim in token")
	} else if sub != expectedSubject {
		return status.Errorf(codes.InvalidArgument, "unexpected subject %s", sub)
	}

	// Now, create the token review
	review := &v1.TokenReview{
		Spec: v1.TokenReviewSpec{
			Token: token,
			// audience remain empty to indicate API server audience
		},
	}

	reviewWithStatus, err := s.k8sAuthnClient.TokenReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to authenticate with Kubernetes API server: %s", err)
	}

	reviewStatus := reviewWithStatus.Status
	if reviewStatus.Error != "" {
		return status.Errorf(codes.Internal, "failed to review authentication token: %s", reviewStatus.Error)
	}
	if !reviewStatus.Authenticated {
		return status.Error(codes.Unauthenticated, "token not authenticated")
	}
	if reviewStatus.User.Username != expectedSubject {
		return status.Errorf(codes.Internal, "authorized unexpected user %q", reviewStatus.User.Username)
	}

	return nil
}

func (s *service) loadCertsForService(serviceName string) (certPEM, keyPEM string, err error) {
	certFileName := filepath.Join(cacheDir, serviceName+"-cert.pem")
	keyFileName := filepath.Join(cacheDir, serviceName+"-key.pem")

	if allExist, err := fileutils.AllExist(certFileName, keyFileName); err != nil {
		return "", "", status.Error(codes.Internal, "failed to check for existence of certificates")
	} else if !allExist {
		return "", "", status.Errorf(codes.NotFound, "no set of certificates for service %s is available", serviceName)
	}

	certBytes, err := ioutil.ReadFile(certFileName)
	if err != nil {
		return "", "", status.Errorf(codes.Internal, "failed to read certificate file: %s", err)
	}
	keyBytes, err := ioutil.ReadFile(keyFileName)
	if err != nil {
		return "", "", status.Errorf(codes.Internal, "failed to read key file: %s", err)
	}

	return string(certBytes), string(keyBytes), nil
}

func (s *service) FetchCertificate(ctx context.Context, req *sensor.FetchCertificateRequest) (*sensor.FetchCertificateResponse, error) {
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, status.Error(codes.PermissionDenied, "rate limit exceeded for this API")
	}

	token := req.GetServiceAccountToken()
	if token == "" {
		return nil, status.Error(codes.InvalidArgument, "no token specified")
	}

	serviceName := services.ServiceTypeToSlugName(req.GetServiceType())
	if serviceName == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid service type %s", req.GetServiceType())
	}

	expectedSubject := fmt.Sprintf("system:serviceaccount:%s:%s", s.namespace, serviceName)
	if err := s.verifyToken(ctx, token, expectedSubject); err != nil {
		return nil, err
	}

	certPEM, keyPEM, err := s.loadCertsForService(serviceName)
	if err != nil {
		return nil, err
	}

	resp := &sensor.FetchCertificateResponse{
		PemCert: certPEM,
		PemKey:  keyPEM,
	}

	return resp, nil
}
