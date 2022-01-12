package certgen

import (
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/renderer"
	"google.golang.org/grpc/codes"
)

func initializeSecretsWithCACertAndKey() (map[string][]byte, mtls.CA, error) {
	ca, err := mtls.LoadDefaultCA()
	if err != nil {
		return nil, nil, err
	}

	fileMap := make(map[string][]byte)
	certgen.AddCAToFileMap(fileMap, ca)
	return fileMap, ca, nil
}

func writeFile(w http.ResponseWriter, contents []byte, fileName string) {
	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	_, _ = w.Write(contents)
}

func (s *serviceImpl) centralHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "invalid method %s, only POST allowed", r.Method)
		return
	}

	secrets, ca, err := initializeSecretsWithCACertAndKey()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if err := certgen.IssueCentralCert(secrets, ca); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	jwtKey, err := jwt.GetPrivateKeyBytes()
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to read JWT key: %v", err)
		return
	}
	secrets[certgen.JWTKeyDERFileName] = jwtKey
	secrets[certgen.JWTKeyPEMFileName] = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: jwtKey,
	})

	rendered, err := renderer.RenderCentralTLSSecretOnly(renderer.Config{
		K8sConfig:      &renderer.K8sConfig{},
		SecretsByteMap: secrets,
	}, defaults.GetImageFlavorFromEnv())
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to render central TLS file: %v", err)
		return
	}

	writeFile(w, rendered, "central-tls.yaml")
}

func (s *serviceImpl) scannerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "invalid method %s, only POST allowed", r.Method)
		return
	}

	secrets, ca, err := initializeSecretsWithCACertAndKey()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if err := certgen.IssueScannerCerts(secrets, ca); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	rendered, err := renderer.RenderScannerTLSSecretOnly(renderer.Config{
		K8sConfig:      &renderer.K8sConfig{},
		SecretsByteMap: secrets,
	}, defaults.GetImageFlavorFromEnv())
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "failed to render scanner TLS file: %v", err)
		return
	}

	writeFile(w, rendered, "scanner-tls.yaml")
}
