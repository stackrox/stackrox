package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stackrox/rox/pkg/uuid"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numServiceCertDataEntries = 3 // cert pem + key pem + ca pem
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralTLS, client)
}

func reconcileCentralTLS(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), log logr.Logger) error {
	run := &createCentralTLSExtensionRun{
		SecretReconciliationExtension: commonExtensions.NewSecretReconciliationExtension(ctx, client, c),
		centralObj:                    c,
		ctx:                           ctx,
	}
	return run.Execute()
}

type createCentralTLSExtensionRun struct {
	*commonExtensions.SecretReconciliationExtension

	ca         mtls.CA
	centralObj *platform.Central
	ctx        context.Context
}

func (r *createCentralTLSExtensionRun) Execute() error {
	shouldDelete := r.centralObj.DeletionTimestamp != nil

	// If we find a broken central-tls secret, do NOT try to auto-fix it. Doing so would invalidate all previously issued certificates
	// (including sensor certificates and init bundles), and is very unlikely to result in a working state.
	if err := r.ReconcileSecret("central-tls", !shouldDelete, r.validateAndConsumeCentralTLSData, r.generateCentralTLSData, false); err != nil {
		return errors.Wrap(err, "reconciling central-tls secret")
	}

	// scanner and scanner-db certs can be re-issued without a problem.
	scannerEnabled := r.centralObj.Spec.Scanner.IsEnabled()
	if err := r.ReconcileSecret("scanner-tls", scannerEnabled && !shouldDelete, r.validateScannerTLSData, r.generateScannerTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner secret")
	}
	if err := r.ReconcileSecret("scanner-db-tls", scannerEnabled && !shouldDelete, r.validateScannerDBTLSData, r.generateScannerDBTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner-db secret")
	}

	bundleSecretShouldExist, err := r.shouldBundleSecretsExist(shouldDelete)
	if err != nil {
		return err
	}
	fixExistingInitBundleSecret := true
	for _, serviceType := range centralsensor.AllSecuredClusterServices {
		slugCaseService := services.ServiceTypeToSlugName(serviceType)
		secretName := slugCaseService + "-tls"
		validateFunc := func(fileMap secretDataMap, _ bool) error {
			return r.validateServiceTLSData(serviceType, slugCaseService+"-", fileMap)
		}
		generateFunc := func() (secretDataMap, error) {
			return r.generateInitBundleTLSData(slugCaseService+"-", serviceType)
		}
		if err := r.ReconcileSecret(secretName, bundleSecretShouldExist, validateFunc, generateFunc, fixExistingInitBundleSecret); err != nil {
			return errors.Wrapf(err, "reconciling %s secret ", slugCaseService)
		}
	}

	return nil
}

func (r *createCentralTLSExtensionRun) shouldBundleSecretsExist(shouldDelete bool) (bool, error) {
	if shouldDelete {
		// Don't bother listing secured clusters if we're ensuring absence of bundle for other reasons.
		return false, nil
	}
	securedClusterPresent, err := r.isSiblingSecuredClusterPresent()
	if err != nil {
		return false, errors.Wrap(err, "determining whether to create init bundle failed")
	}
	return securedClusterPresent, nil
}

func (r *createCentralTLSExtensionRun) validateAndConsumeCentralTLSData(fileMap secretDataMap, _ bool) error {
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

func (r *createCentralTLSExtensionRun) generateCentralTLSData() (secretDataMap, error) {
	var err error
	r.ca, err = certgen.GenerateCA()
	if err != nil {
		return nil, errors.Wrap(err, "creating new CA")
	}

	fileMap := make(secretDataMap)
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

func (r *createCentralTLSExtensionRun) validateServiceTLSData(serviceType storage.ServiceType, fileNamePrefix string, fileMap secretDataMap) error {
	if err := certgen.VerifyServiceCert(fileMap, r.ca, serviceType, fileNamePrefix); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateServiceTLSData(subj mtls.Subject, fileNamePrefix string, fileMap secretDataMap, opts ...mtls.IssueCertOption) error {
	allOpts := append([]mtls.IssueCertOption{mtls.WithNamespace(r.Namespace())}, opts...)
	if err := certgen.IssueServiceCert(fileMap, r.ca, subj, fileNamePrefix, allOpts...); err != nil {
		return err
	}
	certgen.AddCACertToFileMap(fileMap, r.ca)
	return nil
}

func (r *createCentralTLSExtensionRun) validateScannerTLSData(fileMap secretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerTLSData() (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateScannerDBTLSData(fileMap secretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_DB_SERVICE, "", fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerDBTLSData() (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerDBSubject, "", fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) generateInitBundleTLSData(fileNamePrefix string, serviceType storage.ServiceType) (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	bundleID := uuid.NewV4()
	subject := mtls.NewInitSubject(centralsensor.EphemeralInitCertClusterID, serviceType, bundleID)
	if err := r.generateServiceTLSData(subject, fileNamePrefix, fileMap, mtls.WithValidityExpiringInHours()); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) isSiblingSecuredClusterPresent() (bool, error) {
	list := &platform.SecuredClusterList{}
	namespace := r.centralObj.GetNamespace()
	if err := r.Client().List(r.ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list securedclusters in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil

}
