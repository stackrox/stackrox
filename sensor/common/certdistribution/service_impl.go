package certdistribution

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stackrox/rox/sensor/common/clusterid"
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
	maxQueryRate rate.Limit = 1.0

	maxBurstRequests = 10
)

var (
	authorizer = allow.Anonymous() // allow anonymous access because we verify tokens directly with the API server
)

type service struct {
	sensor.UnimplementedCertDistributionServiceServer

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

func (s *service) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	return nil
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) verifyToken(ctx context.Context, token string, expectedSubject string) error {
	parsedToken, err := jwt.ParseSigned(token)
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "invalid JWT: %s", err)
	}

	var claims map[string]interface{}
	if err := parsedToken.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "unparseable claims in token: %s", err)
	}

	if sub, ok := claims["sub"].(string); !ok {
		return errors.Wrap(errox.InvalidArgs, "non-string subject claim in token")
	} else if sub != expectedSubject {
		return errors.Wrapf(errox.InvalidArgs, "unexpected subject %s", sub)
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
		return errors.Wrapf(errox.InvalidArgs, "failed to authenticate with Kubernetes API server: %s", err)
	}

	reviewStatus := reviewWithStatus.Status
	if reviewStatus.Error != "" {
		return errors.Errorf("failed to review authentication token: %s", reviewStatus.Error)
	}
	if !reviewStatus.Authenticated {
		return status.Error(codes.Unauthenticated, "token not authenticated")
	}
	if reviewStatus.User.Username != expectedSubject {
		return errors.Errorf("authorized unexpected user %q", reviewStatus.User.Username)
	}

	return nil
}

func (s *service) loadCertsForService(serviceName string) (certPEM, keyPEM string, err error) {
	certFileName := filepath.Join(cacheDir.Setting(), serviceName+"-cert.pem")
	keyFileName := filepath.Join(cacheDir.Setting(), serviceName+"-key.pem")

	if allExist, err := fileutils.AllExist(certFileName, keyFileName); err != nil {
		return "", "", errors.New("failed to check for existence of certificates")
	} else if !allExist {
		return "", "", errors.Wrapf(errox.NotFound, "no set of certificates for service %s is available", serviceName)
	}

	certBytes, err := os.ReadFile(certFileName)
	if err != nil {
		return "", "", errors.Errorf("failed to read certificate file: %s", err)
	}
	keyBytes, err := os.ReadFile(keyFileName)
	if err != nil {
		return "", "", errors.Errorf("failed to read key file: %s", err)
	}

	return string(certBytes), string(keyBytes), nil
}

func (s *service) verifyRequestViaIdentity(requestingServiceIdentity *storage.ServiceIdentity, serviceType storage.ServiceType) bool {
	if requestingServiceIdentity.GetType() != serviceType {
		return false
	}
	// The following call will return an error if the explicit ID `clusterid.Get()` (which is always a non-wildcard
	// id) is incompatible with the ID from cert `requestingServiceIdentity.GetId()`. In effect, the IDs need
	// to be equal, or the latter (but not the former) needs to be a wildcard ID.
	if _, err := centralsensor.GetClusterID(clusterid.Get(), requestingServiceIdentity.GetId()); err != nil {
		return false
	}
	return true
}

func (s *service) verifyRequestViaServiceAccountToken(ctx context.Context, serviceName, token string) error {
	if token == "" {
		return errors.Wrap(errox.InvalidArgs, "no token specified")
	}

	// This API is rate limit such that an untrusted user cannot get us to DDoS the Kubernetes API server.
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return status.Error(codes.PermissionDenied, "rate limit exceeded for this API")
	}

	expectedSubject := fmt.Sprintf("system:serviceaccount:%s:%s", s.namespace, serviceName)
	if err := s.verifyToken(ctx, token, expectedSubject); err != nil {
		return err
	}
	return nil
}

func (s *service) FetchCertificate(ctx context.Context, req *sensor.FetchCertificateRequest) (*sensor.FetchCertificateResponse, error) {
	serviceName := services.ServiceTypeToSlugName(req.GetServiceType())
	if serviceName == "" {
		return nil, errors.Wrapf(errox.InvalidArgs, "invalid service type %s", req.GetServiceType())
	}

	var requestingServiceIdentity *storage.ServiceIdentity
	if id := authn.IdentityFromContextOrNil(ctx); id != nil {
		requestingServiceIdentity = id.Service()
	}
	// If the request is made with a valid service cert with a matching type, we do not need to go through the
	// Kubernetes API server for verification. This is the case, for example, if the client has a valid cert,
	// which is however not usable for the namespace it runs in.
	if s.verifyRequestViaIdentity(requestingServiceIdentity, req.GetServiceType()) {
		if err := s.verifyRequestViaServiceAccountToken(ctx, serviceName, req.GetServiceAccountToken()); err != nil {
			return nil, err
		}
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
