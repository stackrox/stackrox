package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"k8s.io/client-go/kubernetes"
)

const (
	numServiceCertDataEntries = 3 // cert pem + key pem + ca pem
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(k8sClient kubernetes.Interface) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralTLS, k8sClient)
}

func reconcileCentralTLS(ctx context.Context, c *centralv1Alpha1.Central, k8sClient kubernetes.Interface, log logr.Logger) error {
	run := &createCentralTLSExtensionRun{
		secretReconciliationExtension: secretReconciliationExtension{
			ctx:        ctx,
			centralObj: c,
			k8sClient:  k8sClient,
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

	if err := r.reconcileSecret("central-tls", !shouldDelete, r.validateAndConsumeCentralTLSData, r.generateCentralTLSData); err != nil {
		return errors.Wrap(err, "reconciling central-tls secret")
	}

	scannerEnabled := r.centralObj.Spec.Scanner.IsEnabled()
	if err := r.reconcileSecret("scanner-tls", scannerEnabled && !shouldDelete, r.validateScannerTLSData, r.generateScannerTLSData); err != nil {
		return errors.Wrap(err, "reconciling scanner secret")
	}
	if err := r.reconcileSecret("scanner-db-tls", scannerEnabled && !shouldDelete, r.validateScannerDBTLSData, r.generateScannerDBTLSData); err != nil {
		return errors.Wrap(err, "reconciling scanner-db secret")
	}

	return nil
}

func (r *createCentralTLSExtensionRun) validateAndConsumeCentralTLSData(fileMap secretDataMap) error {
	var err error
	r.ca, err = certgen.LoadCAFromFileMap(fileMap)
	if err != nil {
		return errors.Wrap(err, "loading CA")
	}
	if err := r.ca.CheckProperties(); err != nil {
		return errors.Wrap(err, "loaded service CA certificate is invalid")
	}
	if err := certgen.VerifyServiceCert(fileMap, r.ca, mtls.CentralSubject, ""); err != nil {
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

	if err := certgen.IssueCentralCert(fileMap, r.ca); err != nil {
		return nil, errors.Wrap(err, "issuing central service certificate")
	}

	jwtKey, err := certgen.GenerateJWTSigningKey()
	if err != nil {
		return nil, errors.Wrap(err, "generating JWT signing key")
	}
	certgen.AddJWTSigningKeyToFileMap(fileMap, jwtKey)

	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateServiceTLSData(subj mtls.Subject, fileMap secretDataMap) error {
	if err := certgen.VerifyServiceCert(fileMap, r.ca, subj, ""); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateServiceTLSData(subj mtls.Subject, fileMap secretDataMap) error {
	if err := certgen.IssueServiceCert(fileMap, r.ca, subj, ""); err != nil {
		return err
	}
	certgen.AddCACertToFileMap(fileMap, r.ca)
	return nil
}

func (r *createCentralTLSExtensionRun) validateScannerTLSData(fileMap secretDataMap) error {
	return r.validateServiceTLSData(mtls.ScannerSubject, fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerTLSData() (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerSubject, fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateScannerDBTLSData(fileMap secretDataMap) error {
	return r.validateServiceTLSData(mtls.ScannerDBSubject, fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerDBTLSData() (secretDataMap, error) {
	fileMap := make(secretDataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerDBSubject, fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}
