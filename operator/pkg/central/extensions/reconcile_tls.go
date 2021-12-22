package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	numServiceCertDataEntries = 3 // cert pem + key pem + ca pem
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(client client.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralTLS, client)
}

func reconcileCentralTLS(ctx context.Context, c *platform.Central, client client.Client, _ func(updateStatusFunc), log logr.Logger) error {
	run := &createCentralTLSExtensionRun{
		secretReconciliationExtension: secretReconciliationExtension{
			ctx:        ctx,
			centralObj: c,
			ctrlClient: client,
		},
	}
	return run.Execute()
}

type createCentralTLSExtensionRun struct {
	secretReconciliationExtension

	ca mtls.CA
}

func (r *createCentralTLSExtensionRun) Execute() error {
	shouldDelete := r.centralObj.DeletionTimestamp != nil

	// If we find a broken central-tls secret, do NOT try to auto-fix it. Doing so would invalidate all previously issued certificates
	// (including sensor certificates and init bundles), and is very unlikely to result in a working state.
	if err := r.reconcileSecret("central-tls", !shouldDelete, r.validateAndConsumeCentralTLSData, r.generateCentralTLSData, false); err != nil {
		return errors.Wrap(err, "reconciling central-tls secret")
	}

	// scanner and scanner-db certs can be re-issued without a problem.
	scannerEnabled := r.centralObj.Spec.Scanner.IsEnabled()
	if err := r.reconcileSecret("scanner-tls", scannerEnabled && !shouldDelete, r.validateScannerTLSData, r.generateScannerTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner secret")
	}
	if err := r.reconcileSecret("scanner-db-tls", scannerEnabled && !shouldDelete, r.validateScannerDBTLSData, r.generateScannerDBTLSData, true); err != nil {
		return errors.Wrap(err, "reconciling scanner-db secret")
	}

	return nil
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

func (r *createCentralTLSExtensionRun) validateServiceTLSData(serviceType storage.ServiceType, fileMap secretDataMap) error {
	if err := certgen.VerifyServiceCert(fileMap, r.ca, serviceType, ""); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateServiceTLSData(subj mtls.Subject, fileMap secretDataMap) error {
	if err := certgen.IssueServiceCert(fileMap, r.ca, subj, "", mtls.WithNamespace(r.Namespace())); err != nil {
		return err
	}
	certgen.AddCACertToFileMap(fileMap, r.ca)
	return nil
}

func (r *createCentralTLSExtensionRun) validateScannerTLSData(fileMap secretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_SERVICE, fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerTLSData() (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerSubject, fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateScannerDBTLSData(fileMap secretDataMap, _ bool) error {
	return r.validateServiceTLSData(storage.ServiceType_SCANNER_DB_SERVICE, fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerDBTLSData() (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerDBSubject, fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}
