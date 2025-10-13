package certgen

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/renderer"
	"google.golang.org/grpc/codes"
)

func (s *serviceImpl) centralDBHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "invalid method %s, only POST allowed", r.Method)
		return
	}
	if pgconfig.IsExternalDatabase() {
		httputil.WriteGRPCStyleError(w, codes.FailedPrecondition,
			errors.New("Cannot use this service to generate central DB certificate when using external database"))
	}

	secrets, ca, err := initializeSecretsWithCACertAndKey()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if err := certgen.IssueOtherServiceCerts(secrets, ca, []mtls.Subject{mtls.CentralDBSubject}); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	rendered, err := renderer.RenderCentralDBTLSSecretOnly(renderer.Config{
		K8sConfig:      &renderer.K8sConfig{},
		SecretsByteMap: secrets,
	}, defaults.GetImageFlavorFromEnv())
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to render central-db TLS file: %v", err)
		return
	}

	writeFile(w, rendered, "central-db-tls.yaml")
}
