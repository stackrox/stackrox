package testutils

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	appsApiv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func SensorOwnerReference(sensorDeployment *appsApiv1.Deployment) []metav1.OwnerReference {
	sensorDeploymentGVK := sensorDeployment.GroupVersionKind()
	blockOwnerDeletion := false
	isController := false
	return []metav1.OwnerReference{
		{
			APIVersion:         sensorDeploymentGVK.GroupVersion().String(),
			Kind:               sensorDeploymentGVK.Kind,
			Name:               sensorDeployment.GetName(),
			UID:                sensorDeployment.GetUID(),
			BlockOwnerDeletion: &blockOwnerDeletion,
			Controller:         &isController,
		},
	}
}

func CreateServiceCertificate(serviceType storage.ServiceType) *storage.TypedServiceCertificate {
	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: make([]byte, 0),
			KeyPem:  make([]byte, 1),
		},
	}
}

func IssueCertificate(serviceType storage.ServiceType, issueOption mtls.IssueCertOption) (*mtls.IssuedCert, error) {
	ca, err := mtls.CAForSigning()
	if err != nil {
		return nil, err
	}
	subject := mtls.NewSubject("clusterId", serviceType)
	cert, err := ca.IssueCertForSubject(subject, issueOption)
	if err != nil {
		return nil, err
	}
	return cert, err
}

func IssueCertificatePEM(issueOption mtls.IssueCertOption) ([]byte, error) {
	cert, err := IssueCertificate(storage.ServiceType_SCANNER_SERVICE, issueOption)
	if err != nil {
		return nil, err
	}
	return cert.CertPEM, nil
}
