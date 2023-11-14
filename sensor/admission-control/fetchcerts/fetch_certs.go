package fetchcerts

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/admission-control/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	attemptTimeout = 10 * time.Second
	retryDelay     = 5 * time.Second
)

var (
	certDir  = filepath.Join(common.TempStoragePath, ".certificates")
	certFile = filepath.Join(certDir, "cert.pem")
	keyFile  = filepath.Join(certDir, "key.pem")

	log = logging.LoggerForModule()
)

func changeCertAndKeyFileEnvVars() error {
	if err := utils.ShouldErr(os.Setenv(mtls.CertFilePathEnvName, certFile)); err != nil {
		return errors.Wrap(err, "updating certificate path environment variable")
	}
	if err := utils.ShouldErr(os.Setenv(mtls.KeyFileEnvName, keyFile)); err != nil {
		return errors.Wrap(err, "updating key path environment variable")
	}
	return nil
}

func applyFetchedCertSettings(fetchResult *sensor.FetchCertificateResponse) error {
	if err := os.WriteFile(certFile, []byte(fetchResult.GetPemCert()), 0600); err != nil {
		return errors.Wrap(err, "writing certificate to file")
	}
	if err := os.WriteFile(keyFile, []byte(fetchResult.GetPemKey()), 0600); err != nil {
		return errors.Wrap(err, "writing private key to file")
	}

	return changeCertAndKeyFileEnvVars()
}

func fetchCertificateFromSensor(ctx context.Context, token string) (*sensor.FetchCertificateResponse, error) {
	req := &sensor.FetchCertificateRequest{
		ServiceType:         storage.ServiceType_ADMISSION_CONTROL_SERVICE,
		ServiceAccountToken: token,
	}

	caPool, err := verifier.TrustedCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get service CA verifier")
	}

	conn, err := clientconn.GRPCConnection(ctx, mtls.SensorSubject, env.SensorEndpoint.Setting(), clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			UseClientCert: clientconn.UseClientCertIfAvailable,
			RootCAs:       caPool,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "establishing gRPC connection to sensor")
	}
	defer utils.IgnoreError(conn.Close)

	certDistClient := sensor.NewCertDistributionServiceClient(conn)
	var fetchResult *sensor.FetchCertificateResponse

	err = retry.WithRetry(
		func() error {
			ctx, cancel := context.WithTimeout(ctx, attemptTimeout)
			defer cancel()

			fetchResult, err = certDistClient.FetchCertificate(ctx, req)
			if err != nil {
				spb, ok := status.FromError(err)
				// Only retry unavailable, deadline exceeded, and not found errors. These might resolve over time.
				if ok && (spb.Code() == codes.Unavailable || spb.Code() == codes.DeadlineExceeded || spb.Code() == codes.NotFound) {
					return retry.MakeRetryable(err)
				}
				return err
			}
			return nil
		},
		retry.OnlyRetryableErrors(),
		retry.Tries(10),
		retry.OnFailedAttempts(func(err error) {
			log.Errorf("Failed to fetch certificates from sensor: %s. Retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
		}))
	if err != nil {
		return nil, err
	}
	return fetchResult, nil
}

// FetchAndSetupCertificates attempts to fetch certificates from Sensor, and if successful, sets up the environment
// to read these certs.
func FetchAndSetupCertificates(ctx context.Context) error {
	if allExist, err := fileutils.AllExist(certFile, keyFile); err != nil {
		log.Error("Failed to stat certificates in cached location. Assuming they don't exist")
	} else if allExist {
		log.Info("Reusing cached certificates from previous run")
		return changeCertAndKeyFileEnvVars()
	}

	token, err := satoken.LoadTokenFromFile()
	if err != nil {
		return errors.Wrap(err, "failed to load service account token")
	}

	if err := os.MkdirAll(certDir, 0700); err != nil {
		return errors.Wrap(err, "failed to create certificate cache directory")
	}

	fetchResult, err := fetchCertificateFromSensor(ctx, token)
	if err != nil {
		return errors.Wrap(err, "failed to fetch certificates from sensor")
	}

	if err := applyFetchedCertSettings(fetchResult); err != nil {
		return err
	}

	return nil
}
