package extensions

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/carotation"
	"github.com/stackrox/rox/operator/internal/common"
	"github.com/stackrox/rox/operator/internal/common/confighash"
	commonExtensions "github.com/stackrox/rox/operator/internal/common/extensions"
	commonLabels "github.com/stackrox/rox/operator/internal/common/labels"
	"github.com/stackrox/rox/operator/internal/common/rendercache"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numServiceCertDataEntries = 3 // cert pem + key pem + ca pem

	clusterRegistrationSecretName = "cluster-registration-secret"
	crsDataKey                    = "crs"
	crsAnnotationPrefix           = "crs.platform.stackrox.io/"
	operatorManagedCRSName        = "operator-managed"
	// Keep in sync with mtls.ephemeralProfileWithExpirationInHoursCertLifetime.
	crsCertLifetime = 3 * time.Hour

	// CRSReconcilePeriod is the maximum period between untriggered reconciliations that renew
	// the operator-managed cluster-registration-secret. It must be less than half of
	// crypto.ephemeralProfileWithExpirationInHoursCertLifetime so CRS is renewed before half-validity.
	CRSReconcilePeriod = 1 * time.Hour

	envCentralCARotationEnabled = "CENTRAL_CA_ROTATION_ENABLED"
)

var (
	// centralCARotationEnabled is a feature flag for the Central CA rotation feature.
	centralCARotationEnabled = env.RegisterBooleanSetting(envCentralCARotationEnabled, true)
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(client ctrlClient.Client, direct ctrlClient.Reader, renderCache *rendercache.RenderCache) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralTLS, client, direct, renderCache)
}

func reconcileCentralTLS(ctx context.Context, c *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, _ func(updateStatusFunc), _ logr.Logger, renderCache *rendercache.RenderCache) error {
	run := &createCentralTLSExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, direct, c),
		centralObj:          c,
		currentTime:         time.Now(),
		renderCache:         renderCache,
	}

	return run.Execute(ctx)
}

type createCentralTLSExtensionRun struct {
	*commonExtensions.SecretReconciliator

	ca                    mtls.CA // primary CA, used to issue Central-services certificates
	secondaryCA           mtls.CA // secondary CA, for CA rotation support
	caRotationAction      carotation.Action
	centralObj            *platform.Central
	currentTime           time.Time
	extraIssueCertOptions []mtls.IssueCertOption
	renderCache           *rendercache.RenderCache
}

func (r *createCentralTLSExtensionRun) Execute(ctx context.Context) error {
	if r.centralObj.DeletionTimestamp != nil {
		r.renderCache.Delete(r.centralObj)

		for _, prefix := range []string{"central", "central-db", "scanner", "scanner-db", "scanner-v4-matcher", "scanner-v4-indexer", "scanner-v4-db"} {
			if err := r.DeleteSecret(ctx, prefix+"-tls"); err != nil {
				return errors.Wrapf(err, "reconciling %s-tls secret failed", prefix)
			}
		}
		return r.reconcileCRSSecret(ctx, true)
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

	if r.ca != nil {
		// Add the hash of the CA(s) to the render cache for the pod template annotation post renderer.
		caPEM := r.ca.CertPEM()
		if r.secondaryCA != nil {
			// Include secondary CA if present so that pods restart when it's added during CA rotation.
			caPEM = append(caPEM, r.secondaryCA.CertPEM()...)
		}
		r.renderCache.SetCAHash(r.centralObj, confighash.ComputeCAHash(caPEM))
	}

	return r.reconcileCRSSecret(ctx, false)
}

func (r *createCentralTLSExtensionRun) reconcileCRSSecret(ctx context.Context, shouldDelete bool) error {
	if shouldDelete {
		return r.DeleteSecret(ctx, clusterRegistrationSecretName)
	}
	if err := r.EnsureSecret(ctx, clusterRegistrationSecretName, r.validateCRSData, r.generateCRSData, commonLabels.TLSSecretLabels()); err != nil {
		return errors.Wrap(err, "reconciling cluster-registration-secret failed")
	}
	return r.ensureCRSSecretAnnotations(ctx)
}

func (r *createCentralTLSExtensionRun) validateCRSData(data types.SecretDataMap, _ bool) error {
	crsData, ok := data[crsDataKey]
	if !ok || len(crsData) == 0 {
		return errors.New("missing CRS data")
	}
	crsObj, err := crs.DeserializeSecret(string(crsData))
	if err != nil {
		return errors.Wrap(err, "deserializing CRS")
	}
	if len(crsObj.CAs) == 0 {
		return errors.New("missing CA in CRS")
	}
	if !bytes.Equal([]byte(crsObj.CAs[0]), r.ca.CertPEM()) {
		return errors.New("CRS CA does not match current CA")
	}
	cert, err := parseCRSCertificate(crsObj)
	if err != nil {
		return err
	}
	if _, err := r.ca.ValidateAndExtractSubject(cert, mtls.WithCurrentTime(r.currentTime)); err != nil {
		return errors.Wrap(err, "CRS certificate is not signed by current CA")
	}
	subject := mtls.SubjectFromCommonName(cert.Subject.CommonName)
	if subject.ServiceType != storage.ServiceType_REGISTRANT_SERVICE {
		return fmt.Errorf("unexpected service type %v in CRS certificate", subject.ServiceType)
	}
	if subject.Identifier != centralsensor.EphemeralInitCertClusterID {
		return fmt.Errorf("unexpected cluster ID %q in CRS certificate", subject.Identifier)
	}
	return r.checkCertRenewal(cert)
}

func (r *createCentralTLSExtensionRun) generateCRSData(_ types.SecretDataMap) (types.SecretDataMap, error) {
	crsID := uuid.NewV4()
	subject := mtls.NewInitSubject(centralsensor.EphemeralInitCertClusterID, storage.ServiceType_REGISTRANT_SERVICE, crsID)
	opts := append([]mtls.IssueCertOption{
		mtls.WithValidityExpiringInHours(),
		mtls.WithValidityNotBefore(r.currentTime),
		mtls.WithValidityNotAfter(r.currentTime.Add(crsCertLifetime)),
	}, r.extraIssueCertOptions...)
	issuedCert, err := r.ca.IssueCertForSubject(subject, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "issuing CRS certificate failed")
	}
	crsObj := &crs.CRS{
		Version: 1,
		CAs:     []string{string(r.ca.CertPEM())},
		Cert:    string(issuedCert.CertPEM),
		Key:     string(issuedCert.KeyPEM),
	}
	serialized, err := crs.SerializeSecret(crsObj)
	if err != nil {
		return nil, errors.Wrap(err, "serializing CRS")
	}
	return types.SecretDataMap{crsDataKey: []byte(serialized)}, nil
}

func (r *createCentralTLSExtensionRun) ensureCRSSecretAnnotations(ctx context.Context) error {
	secret := &corev1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.centralObj.GetNamespace(), Name: clusterRegistrationSecretName}
	if err := utils.GetWithFallbackToUncached(ctx, r.Client(), r.UncachedClient(), key, secret); err != nil {
		return errors.Wrap(err, "getting cluster-registration-secret for annotation update")
	}
	if !metav1.IsControlledBy(secret, r.centralObj) {
		return nil
	}

	crsObj, err := crs.DeserializeSecret(string(secret.Data[crsDataKey]))
	if err != nil {
		return errors.Wrap(err, "deserializing CRS for annotation update")
	}
	cert, err := parseCRSCertificate(crsObj)
	if err != nil {
		return err
	}

	crsID := ""
	if len(cert.Subject.Organization) > 0 {
		crsID = cert.Subject.Organization[0]
	}

	desiredAnnotations := map[string]string{
		crsAnnotationPrefix + "name":       operatorManagedCRSName,
		crsAnnotationPrefix + "created-at": cert.NotBefore.Format(time.RFC3339Nano),
		crsAnnotationPrefix + "expires-at": cert.NotAfter.Format(time.RFC3339Nano),
		crsAnnotationPrefix + "id":         crsID,
	}

	if secret.Annotations == nil {
		secret.Annotations = make(map[string]string)
	}

	needsUpdate := false
	for annotationKey, annotationValue := range desiredAnnotations {
		if secret.Annotations[annotationKey] != annotationValue {
			secret.Annotations[annotationKey] = annotationValue
			needsUpdate = true
		}
	}
	if !needsUpdate {
		return nil
	}

	return errors.Wrap(r.Client().Update(ctx, secret), "updating cluster-registration-secret annotations")
}

func parseCRSCertificate(crsObj *crs.CRS) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(crsObj.Cert))
	if block == nil {
		return nil, errors.New("failed to decode CRS certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parsing CRS certificate")
	}
	return cert, nil
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

		// Load secondary CA (its presence is optional).
		r.secondaryCA, err = certgen.LoadSecondaryCAFromFileMap(fileMap)
		if err != nil && !errors.Is(err, certgen.ErrNoCACert) {
			return errors.Wrap(err, "loading secondary CA failed")
		}
		if r.secondaryCA != nil {
			if err := r.secondaryCA.CheckProperties(); err != nil {
				return errors.Wrap(err, "loaded secondary CA is invalid")
			}
		}

		var secondaryCACert *x509.Certificate
		if r.secondaryCA != nil {
			secondaryCACert = r.secondaryCA.Certificate()
		}
		r.caRotationAction = carotation.DetermineAction(r.ca.Certificate(), secondaryCACert, r.currentTime)
		if r.caRotationAction != carotation.NoAction {
			return errors.New("CA rotation action needed")
		}
	}

	if err := r.validateServiceTLSData(storage.ServiceType_CENTRAL_SERVICE, "", fileMap); err != nil {
		return errors.Wrap(err, "verifying existing central service TLS certificate failed")
	}
	return nil
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

		if r.caRotationAction != carotation.NoAction {
			r.ca, err = certgen.LoadCAFromFileMap(newFileMap)
			if err != nil {
				return nil, errors.Wrap(err, "reloading new primary CA failed")
			}

			r.secondaryCA, err = certgen.LoadSecondaryCAFromFileMap(newFileMap)
			if err != nil && !errors.Is(err, certgen.ErrNoCACert) {
				return nil, errors.Wrap(err, "loading secondary CA after rotation action failed")
			}
		}
	}

	opts := append(
		[]mtls.IssueCertOption{mtls.WithNamespace(r.centralObj.GetNamespace())},
		r.extraIssueCertOptions...,
	)
	if err := certgen.IssueCentralCert(newFileMap, r.ca, opts...); err != nil {
		return nil, errors.Wrap(err, "issuing central service certificate failed")
	}

	// The JWT key is used to validate API Tokens. Recreating the key will invalidate all existing tokens.
	// For this reason, we only generate a key if one doesn't already exist.
	if oldJWTKey, oldJWTKeyOK := old[certgen.JWTKeyPEMFileName]; oldJWTKeyOK {
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

	if centralCARotationEnabled.BooleanSetting() {
		if err := r.verifySecondaryCACertInFileMap(fileMap); err != nil {
			return err
		}
	}
	return nil
}

func (r *createCentralTLSExtensionRun) verifySecondaryCACertInFileMap(fileMap types.SecretDataMap) error {
	secondaryCACertPEM := fileMap[mtls.SecondaryCACertFileName]
	if r.secondaryCA == nil {
		if len(secondaryCACertPEM) > 0 {
			return errors.New("unexpected secondary CA certificate in file map")
		}
		return nil
	}
	if len(secondaryCACertPEM) == 0 {
		return errors.New("missing secondary CA certificate in file map")
	}
	if !bytes.Equal(secondaryCACertPEM, r.secondaryCA.CertPEM()) {
		return errors.New("mismatching secondary CA certificate in file map")
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

	if centralCARotationEnabled.BooleanSetting() && r.secondaryCA != nil {
		certgen.AddSecondaryCACertToFileMap(fileMap, r.secondaryCA)
	}
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
