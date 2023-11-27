package extensions

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stackrox/rox/pkg/uuid"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numServiceCertDataEntries = 3 // cert pem + key pem + ca pem
	// InitBundleReconcilePeriod is the maximum period required for reconciliation of an init bundle.
	// It must be sufficient to renew an ephemeral init bundle certificate which has relatively short lifetime (within a matter of hours).
	// NB: keep in sync with crypto.ephemeralProfileWithExpirationInHoursCertLifetime
	InitBundleReconcilePeriod   = 1 * time.Hour
	initBundleGracePeriod       = 90 * time.Minute // half of cert validity period
	fixExistingInitBundleSecret = true
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralTLS, client)
}

func reconcileCentralTLS(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), _ logr.Logger) error {
	run := &createCentralTLSExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, c),
		centralObj:          c,
	}
	return run.Execute(ctx)
}

type createCentralTLSExtensionRun struct {
	*commonExtensions.SecretReconciliator

	ca         mtls.CA
	centralObj *platform.Central
}

func (r *createCentralTLSExtensionRun) Execute(ctx context.Context) error {
	shouldDelete := r.centralObj.DeletionTimestamp != nil

	// If we find a broken central-tls secret, do NOT try to auto-fix it. Doing so would invalidate all previously issued certificates
	// (including sensor certificates and init bundles), and is very unlikely to result in a working state.
	if err := r.ReconcileSecret(ctx, "central-tls", !shouldDelete, r.validateAndConsumeCentralTLSData, r.generateCentralTLSData, false); err != nil {
		return errors.Wrap(err, "reconciling central-tls secret")
	}

	isInternalCentralDB := !r.centralObj.Spec.Central.IsExternalDB()
	if err := r.ReconcileSecret(ctx, "central-db-tls", isInternalCentralDB && !shouldDelete, r.validateCentralDBTLSData, r.generateCentralDBTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling central-db-tls secret")
	}

	// scanner and scanner-db certs can be re-issued without a problem.
	scannerEnabled := r.centralObj.Spec.Scanner.IsEnabled()
	if err := r.ReconcileSecret(ctx, "scanner-tls", scannerEnabled && !shouldDelete, r.validateScannerTLSData, r.generateScannerTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner-tls secret")
	}
	if err := r.ReconcileSecret(ctx, "scanner-db-tls", scannerEnabled && !shouldDelete, r.validateScannerDBTLSData, r.generateScannerDBTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner-db-tls secret")
	}

	if features.ScannerV4.Enabled() {
		scannerV4Enabled := false // TODO: r.centralObj.Spec.Scanner.IsEnabled()
		if err := r.ReconcileSecret(ctx, "scanner-v4-indexer-tls", scannerV4Enabled && !shouldDelete, r.validateScannerV4IndexerTLSData, r.generateScannerV4IndexerTLSData, true); err != nil {
			return errors.Wrap(err, "reconciling scanner-v4-indexer-tls secret")
		}
		if err := r.ReconcileSecret(ctx, "scanner-v4-matcher-tls", scannerV4Enabled && !shouldDelete, r.validateScannerV4MatcherTLSData, r.generateScannerV4MatcherTLSData, true); err != nil {
			return errors.Wrap(err, "reconciling scanner-v4-matcher-tls secret")
		}
		if err := r.ReconcileSecret(ctx, "scanner-v4-db-tls", scannerV4Enabled && !shouldDelete, r.validateScannerV4DBTLSData, r.generateScannerV4DBTLSData, true); err != nil {
			return errors.Wrap(err, "reconciling scanner-v4-db-tls secret")
		}
	}

	return nil // reconcileInitBundleSecrets not called due to ROX-9023. TODO(ROX-9969): call after the init-bundle cert rotation stabilization.
}

//lint:ignore U1000 ignore unused method. TODO(ROX-9969): remove lint ignore after the init-bundle cert rotation stabilization.
func (r *createCentralTLSExtensionRun) reconcileInitBundleSecrets(ctx context.Context, shouldDelete bool) error {
	bundleSecretShouldExist, err := r.shouldBundleSecretsExist(ctx, shouldDelete)
	if err != nil {
		return err
	}
	for _, serviceType := range centralsensor.AllSecuredClusterServices {
		slugCaseService := services.ServiceTypeToSlugName(serviceType)
		secretName := slugCaseService + "-tls"
		validateFunc := func(fileMap types.SecretDataMap, _ bool) error {
			return r.validateInitBundleTLSData(serviceType, slugCaseService+"-", fileMap)
		}
		generateFunc := func() (types.SecretDataMap, error) {
			return r.generateInitBundleTLSData(slugCaseService+"-", serviceType)
		}
		if err := r.ReconcileSecret(ctx, secretName, bundleSecretShouldExist, validateFunc, generateFunc, fixExistingInitBundleSecret); err != nil {
			return errors.Wrapf(err, "reconciling %s secret", secretName)
		}
	}
	return nil
}

func (r *createCentralTLSExtensionRun) shouldBundleSecretsExist(ctx context.Context, shouldDelete bool) (bool, error) {
	if shouldDelete {
		// Don't bother listing secured clusters if we're ensuring absence of bundle for other reasons.
		return false, nil
	}
	securedClusterPresent, err := r.isSiblingSecuredClusterPresent(ctx)
	if err != nil {
		return false, errors.Wrap(err, "determining whether to create init bundle failed")
	}
	return securedClusterPresent, nil
}

func (r *createCentralTLSExtensionRun) validateAndConsumeCentralTLSData(fileMap types.SecretDataMap, _ bool) error {
	var err error
	r.ca, err = certgen.LoadCAFromFileMap(fileMap)
	if err != nil {
		return errors.Wrap(err, "loading CA")
	}
	if err := r.ca.CheckProperties(); err != nil {
		return errors.Wrap(err, "loaded service CA certificate is invalid")
	}
	if err := certgen.VerifyServiceCert(fileMap, r.ca, storage.ServiceType_CENTRAL_SERVICE, ""); err != nil {
		return errors.Wrap(err, "verifying existing central CA")
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateCentralTLSData() (types.SecretDataMap, error) {
	var err error
	r.ca, err = certgen.GenerateCA()
	if err != nil {
		return nil, errors.Wrap(err, "creating new CA")
	}

	fileMap := make(types.SecretDataMap)
	certgen.AddCAToFileMap(fileMap, r.ca)

	if err := certgen.IssueCentralCert(fileMap, r.ca, mtls.WithNamespace(r.Namespace())); err != nil {
		return nil, errors.Wrap(err, "issuing central service certificate")
	}

	jwtKey, err := certgen.GenerateJWTSigningKey()
	if err != nil {
		return nil, errors.Wrap(err, "generating JWT signing key")
	}
	certgen.AddJWTSigningKeyToFileMap(fileMap, jwtKey)

	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateServiceTLSData(serviceType storage.ServiceType, fileNamePrefix string, fileMap types.SecretDataMap) error {
	if err := certgen.VerifyServiceCert(fileMap, r.ca, serviceType, fileNamePrefix); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) validateInitBundleTLSData(serviceType storage.ServiceType, fileNamePrefix string, fileMap types.SecretDataMap) error {
	if err := certgen.VerifyCert(fileMap, fileNamePrefix, r.getValidateInitBundleCert(serviceType)); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) getValidateInitBundleCert(serviceType storage.ServiceType) certgen.ValidateCertFunc {
	validateService := certgen.GetValidateServiceCertFunc(r.ca, serviceType)
	return func(certificate *x509.Certificate) error {
		if err := validateService(certificate); err != nil {
			return err
		}
		if err := checkInitBundleCertRenewal(certificate, time.Now()); err != nil {
			return err
		}
		return nil
	}
}

func checkInitBundleCertRenewal(certificate *x509.Certificate, currentTime time.Time) error {
	startTime := certificate.NotBefore
	if currentTime.Before(startTime) {
		return fmt.Errorf("init bundle secret requires update, certificate lifetime starts in the future, not before: %s", startTime)
	}
	refreshTime := certificate.NotAfter.Add(-initBundleGracePeriod)
	if currentTime.After(refreshTime) {
		return fmt.Errorf("init bundle secret requires update, certificate is expired (or going to expire soon), not after: %s, renew threshold: %s", certificate.NotAfter, refreshTime)
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateServiceTLSData(subj mtls.Subject, fileNamePrefix string, fileMap types.SecretDataMap, opts ...mtls.IssueCertOption) error {
	allOpts := append([]mtls.IssueCertOption{mtls.WithNamespace(r.Namespace())}, opts...)
	if err := certgen.IssueServiceCert(fileMap, r.ca, subj, fileNamePrefix, allOpts...); err != nil {
		return err
	}
	certgen.AddCACertToFileMap(fileMap, r.ca)
	return nil
}

func (r *createCentralTLSExtensionRun) validateScannerTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) validateScannerV4IndexerTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) validateScannerV4MatcherTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_V4_MATCHER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) validateScannerV4DBTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_V4_DB_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerTLSData() (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateScannerV4IndexerTLSData() (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerV4IndexerSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateScannerV4MatcherTLSData() (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerV4MatcherSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateScannerV4DBTLSData() (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerV4DBSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateScannerDBTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_DB_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) validateCentralDBTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_CENTRAL_DB_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerDBTLSData() (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerDBSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateCentralDBTLSData() (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.CentralDBSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateInitBundleTLSData(fileNamePrefix string, serviceType storage.ServiceType) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	bundleID := uuid.NewV4()
	subject := mtls.NewInitSubject(centralsensor.EphemeralInitCertClusterID, serviceType, bundleID)
	if err := r.generateServiceTLSData(subject, fileNamePrefix, fileMap, mtls.WithValidityExpiringInHours()); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) isSiblingSecuredClusterPresent(ctx context.Context) (bool, error) {
	list := &platform.SecuredClusterList{}
	namespace := r.centralObj.GetNamespace()
	if err := r.Client().List(ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list securedclusters in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil
}
