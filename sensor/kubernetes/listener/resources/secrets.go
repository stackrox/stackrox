package resources

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudflare/cfssl/certinfo"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/uuid"
	v1 "k8s.io/api/core/v1"
)

// The following types are copied from the Kubernetes codebase,
// since it is not placed in any of the officially supported client
// libraries.
// dockerConfigJSON represents ~/.docker/config.json file info
// see https://github.com/docker/docker/pull/12009
type dockerConfigJSON struct {
	Auths dockerConfig `json:"auths"`
}

// dockerConfig represents the config file used by the docker CLI.
// This config that represents the credentials that should be used
// when pulling images from specific image repositories.
type dockerConfig map[string]dockerConfigEntry

// dockerConfigEntry is an entry in the dockerConfig.
type dockerConfigEntry struct {
	Username string
	Password string
	Email    string
}

// dockerConfigEntryWithAuth is used solely for deserializing the Auth field
// into a dockerConfigEntry during JSON deserialization.
type dockerConfigEntryWithAuth struct {
	// +optional
	Username string `json:"username,omitempty"`
	// +optional
	Password string `json:"password,omitempty"`
	// +optional
	Email string `json:"email,omitempty"`
	// +optional
	Auth string `json:"auth,omitempty"`
}

// decodeDockerConfigFieldAuth deserializes the "auth" field from dockercfg into a
// username and a password. The format of the auth field is base64(<username>:<password>).
func decodeDockerConfigFieldAuth(field string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(field)
	if err != nil {
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		err = errors.New("unable to parse auth field")
		return
	}

	username = parts[0]
	password = parts[1]

	return
}

func (d *dockerConfigEntry) UnmarshalJSON(data []byte) error {
	var tmp dockerConfigEntryWithAuth
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}

	d.Username = tmp.Username
	d.Password = tmp.Password
	d.Email = tmp.Email

	if len(tmp.Auth) == 0 {
		return nil
	}

	d.Username, d.Password, err = decodeDockerConfigFieldAuth(tmp.Auth)
	return err
}

var dataTypeMap = map[string]storage.SecretType{
	"-----BEGIN CERTIFICATE-----":              storage.SecretType_PUBLIC_CERTIFICATE,
	"-----BEGIN NEW CERTIFICATE REQUEST-----":  storage.SecretType_CERTIFICATE_REQUEST,
	"-----BEGIN PRIVACY-ENHANCED MESSAGE-----": storage.SecretType_PRIVACY_ENHANCED_MESSAGE,
	"-----BEGIN OPENSSH PRIVATE KEY-----":      storage.SecretType_OPENSSH_PRIVATE_KEY,
	"-----BEGIN PGP PRIVATE KEY BLOCK-----":    storage.SecretType_PGP_PRIVATE_KEY,
	"-----BEGIN EC PRIVATE KEY-----":           storage.SecretType_EC_PRIVATE_KEY,
	"-----BEGIN RSA PRIVATE KEY-----":          storage.SecretType_RSA_PRIVATE_KEY,
	"-----BEGIN DSA PRIVATE KEY-----":          storage.SecretType_DSA_PRIVATE_KEY,
	"-----BEGIN PRIVATE KEY-----":              storage.SecretType_CERT_PRIVATE_KEY,
	"-----BEGIN ENCRYPTED PRIVATE KEY-----":    storage.SecretType_ENCRYPTED_PRIVATE_KEY,
}

func getSecretType(data string) storage.SecretType {
	data = strings.TrimSpace(data)
	for dataPrefix, t := range dataTypeMap {
		if strings.HasPrefix(data, dataPrefix) {
			return t
		}
	}
	return storage.SecretType_UNDETERMINED
}

func convertInterfaceSliceToStringSlice(i []interface{}) []string {
	strSlice := make([]string, 0, len(i))
	for _, v := range i {
		strSlice = append(strSlice, fmt.Sprintf("%v", v))
	}
	return strSlice
}

func convertCFSSLName(name certinfo.Name) *storage.CertName {
	return &storage.CertName{
		CommonName:       name.CommonName,
		Country:          name.Country,
		Organization:     name.Organization,
		OrganizationUnit: name.OrganizationalUnit,
		Locality:         name.Locality,
		Province:         name.Province,
		StreetAddress:    name.StreetAddress,
		PostalCode:       name.PostalCode,
		Names:            convertInterfaceSliceToStringSlice(name.Names),
	}
}

func parseCertData(data string) *storage.Cert {
	info, err := certinfo.ParseCertificatePEM([]byte(data))
	if err != nil {
		return nil
	}
	return &storage.Cert{
		Subject:   convertCFSSLName(info.Subject),
		Issuer:    convertCFSSLName(info.Issuer),
		Sans:      info.SANs,
		StartDate: protoconv.ConvertTimeToTimestampOrNil(info.NotBefore),
		EndDate:   protoconv.ConvertTimeToTimestampOrNil(info.NotAfter),
		Algorithm: info.SignatureAlgorithm,
	}
}

func populateTypeData(secret *storage.Secret, dataFiles map[string][]byte) {
	for file, rawData := range dataFiles {
		// Try to base64 decode and if it fails then try the raw value
		var secretType storage.SecretType
		var data string
		decoded, err := base64.StdEncoding.DecodeString(string(rawData))
		if err != nil {
			data = string(rawData)
		} else {
			data = string(decoded)
		}
		secretType = getSecretType(data)

		file := &storage.SecretDataFile{
			Name: file,
			Type: secretType,
		}

		switch secretType {
		case storage.SecretType_PUBLIC_CERTIFICATE:
			file.Metadata = &storage.SecretDataFile_Cert{
				Cert: parseCertData(data),
			}
		}
		secret.Files = append(secret.Files, file)
	}
}

// secretDispatcher handles secret resource events.
type secretDispatcher struct{}

// newSecretDispatcher creates and returns a new secret handler.
func newSecretDispatcher() *secretDispatcher {
	return &secretDispatcher{}
}

func dockerConfigToImageIntegration(registry string, dce dockerConfigEntry) *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:         uuid.NewV4().String(),
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Username: dce.Username,
				Password: dce.Password,
				Insecure: strings.HasPrefix(registry, "http://"),
			},
		},
		Autogenerated: true,
	}
}

func getImageIntegrationSensorEvents(secret *v1.Secret, action central.ResourceAction) []*central.SensorEvent {
	var dockerConfig dockerConfig
	protoSecret := getProtoSecret(secret)
	switch secret.Type {
	case v1.SecretTypeDockercfg:
		data, ok := secret.Data[v1.DockerConfigKey]
		if !ok {
			return nil
		}
		if err := json.Unmarshal(data, &dockerConfig); err != nil {
			log.Error(err)
			return nil
		}
		protoSecret.Files = append(protoSecret.Files, &storage.SecretDataFile{
			Name: v1.DockerConfigKey,
			Type: storage.SecretType_IMAGE_PULL_SECRET,
		})
	case v1.SecretTypeDockerConfigJson:
		data, ok := secret.Data[v1.DockerConfigJsonKey]
		if !ok {
			return nil
		}
		var dockerConfigJSON dockerConfigJSON
		if err := json.Unmarshal(data, &dockerConfigJSON); err != nil {
			log.Error(err)
			return nil
		}
		dockerConfig = dockerConfigJSON.Auths
		protoSecret.Files = append(protoSecret.Files, &storage.SecretDataFile{
			Name: v1.DockerConfigKey,
			Type: storage.SecretType_IMAGE_PULL_SECRET,
		})
	default:
		return nil
	}

	metadata := &storage.SecretDataFile_ImagePullSecret{
		ImagePullSecret: &storage.ImagePullSecret{},
	}

	sensorEvents := make([]*central.SensorEvent, 0, len(dockerConfig))
	registries := make([]*storage.ImagePullSecret_Registry, 0, len(dockerConfig))
	for registry, dce := range dockerConfig {
		ii := dockerConfigToImageIntegration(registry, dce)
		sensorEvents = append(sensorEvents, &central.SensorEvent{
			Action: central.ResourceAction_UPDATE_RESOURCE,
			Resource: &central.SensorEvent_ImageIntegration{
				ImageIntegration: ii,
			},
		})

		registries = append(registries, &storage.ImagePullSecret_Registry{
			Name:     registry,
			Username: dce.Username,
		})
	}
	metadata.ImagePullSecret.Registries = registries
	protoSecret.Files[0].Metadata = metadata

	return append(sensorEvents, secretToSensorEvent(action, protoSecret))
}

func getProtoSecret(secret *v1.Secret) *storage.Secret {
	return &storage.Secret{
		Id:          string(secret.GetUID()),
		Name:        secret.GetName(),
		Namespace:   secret.GetNamespace(),
		Labels:      secret.GetLabels(),
		Annotations: secret.GetAnnotations(),
		CreatedAt:   protoconv.ConvertTimeToTimestamp(secret.GetCreationTimestamp().Time),
	}
}

func secretToSensorEvent(action central.ResourceAction, secret *storage.Secret) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     string(secret.GetId()),
		Action: action,
		Resource: &central.SensorEvent_Secret{
			Secret: secret,
		},
	}
}

// Process processes a secret resource event, and returns the sensor events to emit in response.
func (*secretDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	secret := obj.(*v1.Secret)

	// Filter out service account tokens because we have a service account field.
	// Also filter out DockerConfigJson/DockerCfgs because we don't really care about them.
	switch secret.Type {
	case v1.SecretTypeDockerConfigJson, v1.SecretTypeDockercfg:
		return getImageIntegrationSensorEvents(secret, action)
	case v1.SecretTypeServiceAccountToken:
		return nil
	}

	protoSecret := getProtoSecret(secret)
	populateTypeData(protoSecret, secret.Data)
	return []*central.SensorEvent{
		secretToSensorEvent(action, protoSecret),
	}
}
