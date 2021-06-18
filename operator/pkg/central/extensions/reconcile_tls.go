package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/joelanford/helm-operator/pkg/extensions"
	"github.com/pkg/errors"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/utils"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// dataMap represents data stored as part of a secret.
type dataMap = map[string][]byte

const (
	numServiceCertDataEntries = 3 // cert pem + key pem + ca pem
)

var (
	errUnexpectedGVK = errors.New("invoked reconciliation extension for object with unexpected GVK")
)

// ReconcileCentralTLSExtensions returns an extension that takes care of creating the central-tls and related
// secrets ahead of time.
func ReconcileCentralTLSExtensions(k8sClient kubernetes.Interface) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, log logr.Logger) error {
		return reconcileCentralTLS(ctx, u, k8sClient, log)
	}
}

func reconcileCentralTLS(ctx context.Context, u *unstructured.Unstructured, k8sClient kubernetes.Interface, log logr.Logger) error {
	if u.GroupVersionKind() != centralv1Alpha1.CentralGVK {
		log.Error(errUnexpectedGVK, "unable to reconcile central TLS secrets", "expectedGVK", centralv1Alpha1.CentralGVK, "actualGVK", u.GroupVersionKind())
		return errUnexpectedGVK
	}

	c := centralv1Alpha1.Central{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &c)
	if err != nil {
		return errors.Wrap(err, "converting object to Central")
	}

	namespace := u.GetNamespace()
	secretsClient := k8sClient.CoreV1().Secrets(namespace)

	run := &createCentralTLSExtensionRun{
		ctx:           ctx,
		namespace:     namespace,
		secretsClient: secretsClient,
		centralObj:    &c,
	}
	return run.Execute()
}

type createCentralTLSExtensionRun struct {
	ctx           context.Context
	namespace     string
	secretsClient coreV1.SecretInterface
	centralObj    *centralv1Alpha1.Central

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

func (r *createCentralTLSExtensionRun) reconcileSecret(name string, shouldExist bool, validate func(dataMap) error, generate func() (dataMap, error)) error {
	secret, err := r.secretsClient.Get(r.ctx, name, metav1.GetOptions{})
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if !shouldExist {
		if secret == nil || !metav1.IsControlledBy(secret, r.centralObj) {
			return nil
		}

		if err := utils.DeleteExact(r.ctx, r.secretsClient, secret); err != nil && !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "deleting %s secret", name)
		}
		return nil
	}

	if secret != nil {
		if err := validate(secret.Data); err != nil {
			return errors.Wrapf(err, "validating existing %s secret", name)
		}
		return nil
	}

	data, err := generate()
	if err != nil {
		return errors.Wrapf(err, "generating data for new %s secret", name)
	}
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.centralObj, r.centralObj.GroupVersionKind()),
			},
		},
		Data: data,
	}
	if _, err := r.secretsClient.Create(r.ctx, newSecret, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "creating new %s secret", name)
	}
	return nil
}

func (r *createCentralTLSExtensionRun) validateAndConsumeCentralTLSData(fileMap dataMap) error {
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

func (r *createCentralTLSExtensionRun) generateCentralTLSData() (dataMap, error) {
	var err error
	r.ca, err = certgen.GenerateCA()
	if err != nil {
		return nil, errors.Wrap(err, "creating new CA")
	}

	fileMap := make(dataMap)
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

func (r *createCentralTLSExtensionRun) validateServiceTLSData(subj mtls.Subject, fileMap dataMap) error {
	if err := certgen.VerifyServiceCert(fileMap, r.ca, subj, ""); err != nil {
		return err
	}
	if err := certgen.VerifyCACertInFileMap(fileMap, r.ca); err != nil {
		return err
	}
	return nil
}

func (r *createCentralTLSExtensionRun) generateServiceTLSData(subj mtls.Subject, fileMap dataMap) error {
	if err := certgen.IssueServiceCert(fileMap, r.ca, subj, ""); err != nil {
		return err
	}
	certgen.AddCACertToFileMap(fileMap, r.ca)
	return nil
}

func (r *createCentralTLSExtensionRun) validateScannerTLSData(fileMap dataMap) error {
	return r.validateServiceTLSData(mtls.ScannerSubject, fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerTLSData() (dataMap, error) {
	fileMap := make(dataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerSubject, fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}

func (r *createCentralTLSExtensionRun) validateScannerDBTLSData(fileMap dataMap) error {
	return r.validateServiceTLSData(mtls.ScannerDBSubject, fileMap)
}

func (r *createCentralTLSExtensionRun) generateScannerDBTLSData() (dataMap, error) {
	fileMap := make(dataMap, numServiceCertDataEntries)
	if err := r.generateServiceTLSData(mtls.ScannerDBSubject, fileMap); err != nil {
		return nil, err
	}
	return fileMap, nil
}
