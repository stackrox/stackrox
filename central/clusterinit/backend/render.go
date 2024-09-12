package backend

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sJson "k8s.io/apimachinery/pkg/runtime/serializer/json"
)

const (
	initBundleHeader = `# This is a StackRox cluster init bundle.
# This bundle can be used for setting up any number of StackRox secured clusters.
# NOTE: This file contains secret data and needs to be handled and stored accordingly.
`

	caConfigHeader = `# This is a StackRox CA configuration.
# It can be used with the stackrox-secured-cluster-services Helm chart, provided that
# (a) secrets exist from a previous 'helm install', or (b) secrets have been manually
# pre-created in Kubernetes.`

	initBundleHeaderMeta = `#
#   name:      %q
#   createdAt: %v
#   expiresAt: %v
#   id:        %s
#
`

	// TODO/IDEA: Depending on Branding use ACS or StackRox?
	crsHeader = `# This is a StackRox Cluster Registration Secret (CRS).
# It is used for setting up StackRox secured clusters.
# NOTE: This file contains secret data that allows connecting new secured clusters to central,
# and needs to be handled and stored accordingly.
`
)

// RenderAsYAML renders the CA config as a YAML file that can be used with Helm.
func (c *CAConfig) RenderAsYAML() ([]byte, error) {
	bundleMap := map[string]interface{}{
		"ca": map[string]interface{}{
			"cert": c.CACert,
		},
	}

	var buf bytes.Buffer
	fmt.Fprintln(&buf, caConfigHeader)
	yamlEnc := yaml.NewEncoder(&buf)
	yamlEnc.SetIndent(2)
	if err := yamlEnc.Encode(bundleMap); err != nil {
		return nil, errors.Wrap(err, "encoding CA config to YAML")
	}
	return buf.Bytes(), nil
}

func serviceTLS(cert *mtls.IssuedCert) map[string]interface{} {
	return map[string]interface{}{
		"serviceTLS": map[string]interface{}{
			"cert": string(cert.CertPEM),
			"key":  string(cert.KeyPEM),
		},
	}
}

// RenderAsYAML renders the receiver init bundle as YAML.
func (b *InitBundleWithMeta) RenderAsYAML() ([]byte, error) {
	certBundle := b.CertBundle
	sensorTLS := certBundle[storage.ServiceType_SENSOR_SERVICE]
	if sensorTLS == nil {
		return nil, errors.New("no sensor certificate in init bundle")
	}
	admissionControlTLS := certBundle[storage.ServiceType_ADMISSION_CONTROL_SERVICE]
	if admissionControlTLS == nil {
		return nil, errors.New("no admission control certificate in init bundle")
	}
	collectorTLS := certBundle[storage.ServiceType_COLLECTOR_SERVICE]
	if collectorTLS == nil {
		return nil, errors.New("no collector certificate in init bundle")
	}

	bundleMap := map[string]interface{}{
		"ca": map[string]interface{}{
			"cert": b.CACert,
		},
		"sensor":           serviceTLS(sensorTLS),
		"collector":        serviceTLS(collectorTLS),
		"admissionControl": serviceTLS(admissionControlTLS),
	}

	var bundleBuffer bytes.Buffer

	fmt.Fprint(&bundleBuffer, initBundleHeader)
	fmt.Fprintf(&bundleBuffer,
		initBundleHeaderMeta,
		b.Meta.GetName(),
		b.Meta.GetCreatedAt(),
		b.Meta.GetExpiresAt(),
		b.Meta.GetId())

	yamlEnc := yaml.NewEncoder(&bundleBuffer)
	if err := yamlEnc.Encode(bundleMap); err != nil {
		return nil, errors.Wrap(err, "YAML marshalling of init bundle")
	}

	return bundleBuffer.Bytes(), nil
}

// RenderAsK8sSecrets renders the given init bundle as a list of Kubernetes secrets.
func (b *InitBundleWithMeta) RenderAsK8sSecrets() ([]byte, error) {
	yamlSerializer := k8sJson.NewSerializerWithOptions(
		k8sJson.DefaultMetaFactory, nil, nil, k8sJson.SerializerOptions{Yaml: true})

	var buf bytes.Buffer
	_, _ = fmt.Fprint(&buf, initBundleHeader)
	_, _ = fmt.Fprintf(&buf,
		initBundleHeaderMeta,
		b.Meta.GetName(),
		b.Meta.GetCreatedAt(),
		b.Meta.GetExpiresAt(),
		b.Meta.GetId())
	_, _ = fmt.Fprintln(&buf)

	first := true
	for svcType, cert := range b.CertBundle {
		if first {
			first = false
		} else {
			_, _ = fmt.Fprintln(&buf, "---")
		}

		serviceTypeStr := strings.ToLower(
			strings.ReplaceAll(strings.TrimSuffix(svcType.String(), "_SERVICE"), "_", "-"))
		secret := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceTypeStr + "-tls",
				Annotations: map[string]string{
					"init-bundle.stackrox.io/name":       b.Meta.GetName(),
					"init-bundle.stackrox.io/created-at": b.Meta.GetCreatedAt().AsTime().Format(time.RFC3339Nano),
					"init-bundle.stackrox.io/expires-at": b.Meta.GetExpiresAt().AsTime().Format(time.RFC3339Nano),
					"init-bundle.stackrox.io/id":         b.Meta.GetId(),
				},
			},
			StringData: map[string]string{
				mtls.CACertFileName:                             b.CACert,
				serviceTypeStr + "-" + mtls.ServiceCertFileName: string(cert.CertPEM),
				serviceTypeStr + "-" + mtls.ServiceKeyFileName:  string(cert.KeyPEM),
			},
		}

		if err := yamlSerializer.Encode(secret, &buf); err != nil {
			return nil, errors.Wrapf(err, "encoding secret for service %s", serviceTypeStr)
		}
	}

	return buf.Bytes(), nil
}

// RenderAsK8sSecret renders the given CRS as a Kubernetes secret.
func (b *CRSWithMeta) RenderAsK8sSecret() ([]byte, error) {
	yamlSerializer := k8sJson.NewSerializerWithOptions(
		k8sJson.DefaultMetaFactory, nil, nil, k8sJson.SerializerOptions{Yaml: true})

	var buf bytes.Buffer
	_, _ = fmt.Fprint(&buf, crsHeader)

	crs, err := serializeSecret(b.CRS)
	if err != nil {
		return nil, errors.Wrap(err, "serializing CRS")
	}

	svcType := storage.ServiceType_REGISTRANT_SERVICE
	serviceTypeStr := strings.ToLower(
		strings.ReplaceAll(strings.TrimSuffix(svcType.String(), "_SERVICE"), "_", "-"))
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster-registration-secret",
			Annotations: map[string]string{
				"crs.platform.stackrox.io/name":       b.Meta.GetName(),
				"crs.platform.stackrox.io/created-at": b.Meta.GetCreatedAt().AsTime().Format(time.RFC3339Nano),
				"crs.platform.stackrox.io/expires-at": b.Meta.GetExpiresAt().AsTime().Format(time.RFC3339Nano),
				"crs.platform.stackrox.io/id":         b.Meta.GetId(),
			},
		},
		Data: map[string][]byte{
			"crs": []byte(crs),
		},
	}

	if err := yamlSerializer.Encode(secret, &buf); err != nil {
		return nil, errors.Wrapf(err, "encoding secret for service %s", serviceTypeStr)
	}

	return buf.Bytes(), nil
}

func serializeSecret(crs *CRS) (string, error) {
	jsonSerialized, err := json.Marshal(crs)
	if err != nil {
		return "", errors.Wrap(err, "JSON marshalling CRS")
	}
	base64Encoded := base64.StdEncoding.EncodeToString(jsonSerialized)
	return base64Encoded, nil
}

func deserializeSecret(serializedCrs string) (*CRS, error) {
	var deserializedCrs CRS
	base64Decoded, err := base64.StdEncoding.DecodeString(serializedCrs)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decoding CRS")
	}
	err = json.Unmarshal(base64Decoded, &deserializedCrs)
	if err != nil {
		return nil, errors.Wrap(err, "JSON unmarshalling CRS")
	}
	return &deserializedCrs, nil
}
