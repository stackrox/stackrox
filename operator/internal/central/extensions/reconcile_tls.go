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
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/carotation"
	"github.com/stackrox/rox/operator/internal/common"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	commonLabels "github.com/stackrox/rox/operator/internal/common/labels"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/env"
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
	envCentralCARotationEnabled = "CENTRAL_CA_ROTATION_ENABLED"
)

var (
	// centralCARotationEnabled is a feature flag for the Central CA rotation feature. Defaults to false because
	// the feature is still under active development.
	// TODO: Remove when epic ROX-20262 is complete.
	centralCARotationEnabled = env.RegisterBooleanSetting(envCentralCARotationEnabled, false)
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(client ctrlClient.Client, direct ctrlClient.Reader) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralTLS, client, direct)
}

func reconcileCentralTLS(ctx context.Context, c *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, _ func(updateStatusFunc), _ logr.Logger) error {
	run := &createCentralTLSExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, direct, c),
		centralObj:          c,
		currentTime:         time.Now(),
	}

	return run.Execute(ctx)
}

type createCentralTLSExtensionRun struct {
	*commonExtensions.SecretReconciliator

	ca                    mtls.CA // primary CA, used to issue Central-services certificates
	caRotationAction      carotation.Action
	centralObj            *platform.Central
	currentTime           time.Time
	extraIssueCertOptions []mtls.IssueCertOption
}

func (r *createCentralTLSExtensionRun) Execute(ctx context.Context) error {
	if r.centralObj.DeletionTimestamp != nil {
		for _, prefix := range []string{"central", "central-db", "scanner", "scanner-db", "scanner-v4-matcher", "scanner-v4-indexer", "scanner-v4-db"} {
			if err := r.DeleteSecret(ctx, prefix+"-tls"); err != nil {
				return errors.Wrapf(err, "reconciling %s-tls secret failed", prefix)
			}
		}
		return nil
		// reconcileInitBundleSecrets not called due to ROX-9023. TODO(ROX-9969): call after the init-bundle cert rotation stabilization.
	}

	if err := r.EnsureSecret(ctx, common.CentralTLSSecretName, r.validateAndConsumeCentralTLSData, r.generateCentralTLSData, commonLabels.TLSSecretLabels()); err != nil {
		return errors.Wrap(err, "reconciling central-tls secret failed")
	}

	if err := r.reconcileCentralDBTLSSecret(ctx); err != nil {
		return errors.Wrap(err, "reconciling central-db-tls secret failed")
	}

	if err := r.reconcileScannerTLSSecret(ctx); err != nil {
		return errors.Wrap(err, "reconciling scanner-tls secret failed")
	}
	if err := r.reconcileScannerDBTLSSecret(ctx); err != nil {
		return errors.Wrap(err, "reconciling scanner-db-tls secret failed")
	}

	if err := r.reconcileScannerV4IndexerTLSSecret(ctx); err != nil {
		return errors.Wrap(err, "reconciling scanner-v4-indexer-tls secret")
	}
	if err := r.reconcileScannerV4MatcherTLSSecret(ctx); err != nil {
		return errors.Wrap(err, "reconciling scanner-v4-matcher-tls secret")
	}
	if err := r.reconcileScannerV4DBTLSSecret(ctx); err != nil {
		return errors.Wrap(err, "reconciling scanner-v4-db-tls secret")
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
		if !bundleSecretShouldExist {
			if err := r.DeleteSecret(ctx, secretName); err != nil {
				return errors.Wrapf(err, "deleting %s secret failed", secretName)
			}
			continue
		}
		validateFunc := func(fileMap types.SecretDataMap, _ bool) error {
			return r.validateServiceTLSData(serviceType, slugCaseService+"-", fileMap)
		}
		generateFunc := func(_ types.SecretDataMap) (types.SecretDataMap, error) {
			return r.generateInitBundleTLSData(slugCaseService+"-", serviceType)
		}
		if err := r.EnsureSecret(ctx, secretName, validateFunc, generateFunc, commonLabels.TLSSecretLabels()); err != nil {
			return errors.Wrapf(err, "reconciling %s secret failed", secretName)
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
		return errors.Wrap(err, "loading CA failed")
	}
	if err := r.ca.CheckProperties(); err != nil {
		return errors.Wrap(err, "loaded service CA certificate is invalid")
	}

	if centralCARotationEnabled.BooleanSetting() {
		if err := r.checkCertificateTimeValidity(r.ca.Certificate()); err != nil {
			return errors.Wrap(err, "primary CA is not valid at the present time")
		}

		rotationAction, err := r.determineCARotationAction(fileMap)
		if err != nil {
			return err
		}
		r.caRotationAction = rotationAction
		if r.caRotationAction != carotation.NoAction {
			return errors.New("CA rotation action needed")
		}
	}

	if err := r.validateServiceTLSData(storage.ServiceType_CENTRAL_SERVICE, "", fileMap); err != nil {
		return errors.Wrap(err, "verifying existing central service TLS certificate failed")
	}
	return nil
}

func (r *createCentralTLSExtensionRun) determineCARotationAction(fileMap types.SecretDataMap) (carotation.Action, error) {
	secondaryCA, err := certgen.LoadSecondaryCAFromFileMap(fileMap)
	var secondaryCACert *x509.Certificate
	// the presence of a secondary CA certificate is optional
	if err != nil && !errors.Is(err, certgen.ErrNoCACert) {
		return carotation.NoAction, errors.Wrap(err, "loading secondary CA failed")
	}
	if secondaryCA != nil {
		if err := secondaryCA.CheckProperties(); err != nil {
			return carotation.NoAction, errors.Wrap(err, "loaded secondary service CA certificate is invalid")
		}
		secondaryCACert = secondaryCA.Certificate()
	}

	return carotation.DetermineAction(r.ca.Certificate(), secondaryCACert, r.currentTime), nil
}

func (r *createCentralTLSExtensionRun) generateCentralTLSData(old types.SecretDataMap) (types.SecretDataMap, error) {
	var (
		err        error
		newFileMap types.SecretDataMap
	)
	r.ca, newFileMap, err = validateOrGenerateCA(r.ca, old)
	if err != nil {
		return nil, err
	}

	if centralCARotationEnabled.BooleanSetting() {
		if err = validateSecondaryCA(old, newFileMap); err != nil {
			return nil, err
		}

		if err = carotation.Handle(r.caRotationAction, newFileMap); err != nil {
			return nil, errors.Wrapf(err, "performing CA rotation action: %v", r.caRotationAction)
		}

		if r.caRotationAction == carotation.PromoteSecondary {
			primaryCA, err := certgen.LoadCAFromFileMap(newFileMap)
			if err != nil {
				return nil, errors.Wrap(err, "reloading new primary CA failed")
			}
			r.ca = primaryCA
		}
	}

	opts := append(
		[]mtls.IssueCertOption{mtls.WithNamespace(r.centralObj.GetNamespace())},
		r.extraIssueCertOptions...,
	)
	if err := certgen.IssueCentralCert(newFileMap, r.ca, opts...); err != nil {
		return nil, errors.Wrap(err, "issuing central service certificate failed")
	}

	if oldJWTKey, oldJWTKeyOK := old[certgen.JWTKeyPEMFileName]; oldJWTKeyOK {
		// The impact of replacing the JWT key is unclear.
		// Avoid re-generating JWT key if it exists, out of an abundance of caution.
		// Perhaps this can be changed in the future if we have a way of validating such key.
		newFileMap[certgen.JWTKeyPEMFileName] = oldJWTKey
	} else {
		jwtKey, err := certgen.GenerateJWTSigningKey()
		if err != nil {
			return nil, errors.Wrap(err, "generating JWT signing key failed")
		}
		certgen.AddJWTSigningKeyToFileMap(newFileMap, jwtKey)
	}

	// Since integrity of the central-tls secret is critical to the whole system,
	// we additionally verify it here. Ideally this would be done on the ReconcileSecret level,
	// for all its invocations, but unfortunately some verification functions are currently not idempotent.
	if err := r.validateAndConsumeCentralTLSData(newFileMap, true); err != nil {
		return nil, errors.Wrap(err, "post-generation validation failed")
	}

	return newFileMap, nil
}

func validateOrGenerateCA(oldCA mtls.CA, oldFileMap types.SecretDataMap) (mtls.CA, types.SecretDataMap, error) {
	newFileMap := make(types.SecretDataMap)

	caCert, caCertPresent := oldFileMap[mtls.CACertFileName]
	caKey, caKeyPresent := oldFileMap[mtls.CAKeyFileName]
	if caCertPresent && caKeyPresent {
		// There is an existing CA in the secret. Avoid changing it at all cost, as doing so would immediately cause
		// all previously issued certificates (including sensor certificates and init bundles) to become invalid,
		// and this is very unlikely to result in a working state.
		newFileMap[mtls.CACertFileName] = caCert
		newFileMap[mtls.CAKeyFileName] = caKey
		if oldCA == nil {
			// validateAndConsumeCentralTLSData must have decided the CA is completely unusable.
			// There is not much we can do in this situation, so let's try and provide a useful error message at least.
			_, err := certgen.LoadCAFromFileMap(oldFileMap)
			return nil, nil, errors.Wrap(err, "invalid CA in the existing secret, please delete it to allow re-generation")
		}
		return oldCA, newFileMap, errors.Wrap(oldCA.CheckProperties(), "invalid properties of CA in the existing secret, please delete it to allow re-generation")
	} else if !caCertPresent && !caKeyPresent {
		ca, err := certgen.GenerateCA()
		if err != nil {
			return nil, nil, errors.Wrap(err, "creating new CA failed")
		}
		certgen.AddCAToFileMap(newFileMap, ca)
		return ca, newFileMap, nil
	}
	const msg = "malformed secret (%s present but %s missing), please delete it to allow re-generation"
	if !caCertPresent {
		return nil, nil, fmt.Errorf(msg, mtls.CAKeyFileName, mtls.CACertFileName)
	}
	return nil, nil, fmt.Errorf(msg, mtls.CACertFileName, mtls.CAKeyFileName)
}

func validateSecondaryCA(oldFileMap, newFileMap types.SecretDataMap) error {
	caCert, caCertPresent := oldFileMap[mtls.SecondaryCACertFileName]
	caKey, caKeyPresent := oldFileMap[mtls.SecondaryCAKeyFileName]

	if caCertPresent && caKeyPresent {
		_, err := certgen.LoadSecondaryCAFromFileMap(oldFileMap)
		if err != nil {
			// Secured Clusters might already be using the secondary CA to connect to Central, so re-creating it will
			// not fix things.
			return errors.Wrap(err, "invalid secondary CA in the existing secret, please delete it to allow re-generation")
		}

		newFileMap[mtls.SecondaryCACertFileName] = caCert
		newFileMap[mtls.SecondaryCAKeyFileName] = caKey
		return nil
	} else if caCertPresent || caKeyPresent {
		const msg = "malformed secret (%s present but %s missing), please delete it to allow re-generation"
		if !caCertPresent {
			return fmt.Errorf(msg, mtls.CAKeyFileName, mtls.CACertFileName)
		}
		return fmt.Errorf(msg, mtls.CACertFileName, mtls.CAKeyFileName)
	}

	return nil
}

func (r *createCentralTLSExtensionRun) checkCertificateTimeValidity(certificate *x509.Certificate) error {
	startTime := certificate.NotBefore
	endTime := certificate.NotAfter
	if !endTime.After(startTime) {
		return fmt.Errorf("certificate expires at %s before it begins to be valid at %s", endTime, startTime)
	}
	if r.currentTime.Before(startTime) {
		return fmt.Errorf("certificate lifetime start %s is in the future", startTime)
	}
	if r.currentTime.After(endTime) {
		return fmt.Errorf("certificate expired at %s", endTime)
	}

	return nil
}

func (r *createCentralTLSExtensionRun) reconcileCentralDBTLSSecret(ctx context.Context) error {
	if r.centralObj.Spec.Central.ShouldManageDB() {
		return r.EnsureSecret(ctx, "central-db-tls", r.validateCentralDBTLSData, r.generateCentralDBTLSData, commonLabels.TLSSecretLabels())
	}
	return r.DeleteSecret(ctx, "central-db-tls")
}

func (r *createCentralTLSExtensionRun) reconcileScannerTLSSecret(ctx context.Context) error {
	if r.centralObj.Spec.Scanner.IsEnabled() {
		return r.EnsureSecret(ctx, "scanner-tls", r.validateScannerTLSData, r.generateScannerTLSData, commonLabels.TLSSecretLabels())
	}
	return r.DeleteSecret(ctx, "scanner-tls")
}

func (r *createCentralTLSExtensionRun) reconcileScannerDBTLSSecret(ctx context.Context) error {
	if r.centralObj.Spec.Scanner.IsEnabled() {
		return r.EnsureSecret(ctx, "scanner-db-tls", r.validateScannerDBTLSData, r.generateScannerDBTLSData, commonLabels.TLSSecretLabels())
	}
	return r.DeleteSecret(ctx, "scanner-db-tls")
}

func (r *createCentralTLSExtensionRun) reconcileScannerV4IndexerTLSSecret(ctx context.Context) error {
	if r.centralObj.Spec.ScannerV4.IsEnabled() {
		return r.EnsureSecret(ctx, "scanner-v4-indexer-tls", r.validateScannerV4IndexerTLSData, r.generateScannerV4IndexerTLSData, commonLabels.TLSSecretLabels())
	}
	return r.DeleteSecret(ctx, "scanner-v4-indexer-tls")
}

func (r *createCentralTLSExtensionRun) reconcileScannerV4MatcherTLSSecret(ctx context.Context) error {
	if r.centralObj.Spec.ScannerV4.IsEnabled() {
		return r.EnsureSecret(ctx, "scanner-v4-matcher-tls", r.validateScannerV4MatcherTLSData, r.generateScannerV4MatcherTLSData, commonLabels.TLSSecretLabels())
	}
	return r.DeleteSecret(ctx, "scanner-v4-matcher-tls")
}

func (r *createCentralTLSExtensionRun) reconcileScannerV4DBTLSSecret(ctx context.Context) error {
	if r.centralObj.Spec.ScannerV4.IsEnabled() {
		return r.EnsureSecret(ctx, "scanner-v4-db-tls", r.validateScannerV4DBTLSData, r.generateScannerV4DBTLSData, commonLabels.TLSSecretLabels())
	}
	return r.DeleteSecret(ctx, "scanner-v4-db-tls")
}

func (r *createCentralTLSExtensionRun) validateServiceTLSData(serviceType storage.ServiceType, fileNamePrefix string, fileMap types.SecretDataMap) error {
	if err := certgen.VerifyServiceCertAndKey(fileMap, fileNamePrefix, r.ca, serviceType, &r.currentTime, r.checkCertRenewal); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) checkCertRenewal(certificate *x509.Certificate) error {
	if err := r.checkCertificateTimeValidity(certificate); err != nil {
		return err
	}
	startTime := certificate.NotBefore
	endTime := certificate.NotAfter
	validityDuration := endTime.Sub(startTime)
	halfOfValidityDuration := time.Duration(validityDuration.Nanoseconds()/2) * time.Nanosecond
	refreshTime := startTime.Add(halfOfValidityDuration)
	if r.currentTime.After(refreshTime) {
		return fmt.Errorf("certificate is past half of its validity, %s", refreshTime)
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateServiceTLSData(subj mtls.Subject, fileNamePrefix string, fileMap types.SecretDataMap, opts ...mtls.IssueCertOption) error {
	allOpts := append([]mtls.IssueCertOption{mtls.WithNamespace(r.centralObj.GetNamespace())}, opts...)
	allOpts = append(allOpts, r.extraIssueCertOptions...)
	if err := certgen.IssueServiceCert(fileMap, r.ca, subj, fileNamePrefix, allOpts...); err != nil {
		return err
	}
	certgen.AddCACertToFileMap(fileMap, r.ca)
	return nil
}

func (r *createCentralTLSExtensionRun) validateScannerTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerTLSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerSubject, "", fileMap); err != nil {
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

func (r *createCentralTLSExtensionRun) validateScannerV4IndexerTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) validateScannerV4MatcherTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_V4_MATCHER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) validateScannerV4DBTLSData(fileMap types.SecretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_V4_DB_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerDBTLSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerDBSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateCentralDBTLSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.CentralDBSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateScannerV4IndexerTLSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerV4IndexerSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateScannerV4MatcherTLSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerV4MatcherSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateScannerV4DBTLSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	fileMap := make(types.SecretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerV4DBSubject, "", fileMap); err != nil {
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
