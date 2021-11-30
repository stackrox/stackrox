package resources

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/cloudflare/cfssl/certinfo"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/sensor"
	v1 "k8s.io/api/core/v1"
)

const (
	redhatRegistryEndpoint = "registry.redhat.io"

	// SecretTypeHelmReleaseV1 is where Helm stores the metadata for each
	// release starting with Helm 3.
	// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
	secretTypeHelmReleaseV1 v1.SecretType = "helm.sh/release.v1"
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
	sort.Slice(secret.Files, func(i, j int) bool {
		return secret.Files[i].Name < secret.Files[j].Name
	})
}

// secretDispatcher handles secret resource events.
type secretDispatcher struct{
	sensor              *sensor.Sensor
	// Zero value if not managed by Helm
	helmReleaseName     string
	// Zero value if not managed by Helm
	helmReleaseRevision uint64
}

// newSecretDispatcher creates and returns a new secret handler.
func newSecretDispatcher(sensor *sensor.Sensor, helmManagedConfig *central.HelmManagedConfigInit, deploymentIdentification *storage.SensorDeploymentIdentification) *secretDispatcher {
	return &secretDispatcher{
		sensor: sensor,
		helmReleaseName: helmManagedConfig.HelmReleaseName,
		helmReleaseRevision: deploymentIdentification.HelmReleaseRevision,
	}
}

func dockerConfigToImageIntegration(registry string, dce dockerConfigEntry) *storage.ImageIntegration {
	registryType := docker.GenericDockerRegistryType
	if urlfmt.TrimHTTPPrefixes(registry) == redhatRegistryEndpoint {
		registryType = rhel.RedHatRegistryType
	}

	username, password := dce.Username, dce.Password
	// TODO(ROX-8465): Determine which Service Account's token to use to replace the credentials.
	//if features.LocalImageScanning.Enabled() {
	//}

	return &storage.ImageIntegration{
		Id:         uuid.NewV4().String(),
		Type:       registryType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Username: username,
				Password: password,
			},
		},
		Autogenerated: true,
	}
}

func processDockerConfigEvent(secret *v1.Secret, action central.ResourceAction) []*central.SensorEvent {
	var dockerConfig dockerConfig
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
	default:
		utils.Should(errors.New("only Docker Config secrets are allowed"))
		return nil
	}

	sensorEvents := make([]*central.SensorEvent, 0, len(dockerConfig)+1)
	registries := make([]*storage.ImagePullSecret_Registry, 0, len(dockerConfig))
	for registry, dce := range dockerConfig {
		ii := dockerConfigToImageIntegration(registry, dce)
		sensorEvents = append(sensorEvents, &central.SensorEvent{
			// Only update is supported at this time.
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

	protoSecret := getProtoSecret(secret)
	protoSecret.Files = []*storage.SecretDataFile{{
		Name: v1.DockerConfigKey,
		Type: storage.SecretType_IMAGE_PULL_SECRET,
		Metadata: &storage.SecretDataFile_ImagePullSecret{
			ImagePullSecret: &storage.ImagePullSecret{
				Registries: registries,
			},
		},
	}}

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
		Type: 		 string(secret.Type),
	}
}

func secretToSensorEvent(action central.ResourceAction, secret *storage.Secret) *central.SensorEvent {
	return &central.SensorEvent{
		Id:     secret.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Secret{
			Secret: secret,
		},
	}
}

func (d *secretDispatcher) isHelmManaged() bool {
	return d.helmReleaseRevision > 0 && d.helmReleaseName != ""
}

// ProcessEvent processes a secret resource event, and returns the sensor events to emit in response.
func (d *secretDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	secret := obj.(*v1.Secret)

	switch secret.Type {
	case v1.SecretTypeDockerConfigJson, v1.SecretTypeDockercfg:
		return processDockerConfigEvent(secret, action)
	case v1.SecretTypeServiceAccountToken:
		// Filter out service account tokens because we have a service account field.
		return nil
	}

	if d.isHelmManaged() && isHelmSecret(secret) {
		revision, err := ExtractHelmRevisionFromHelmSecret(d.helmReleaseName, secret)
		if err != nil {
			err := errors.Wrap(err, "failed to extract Helm revision from secret, ignoring potential new Helm release")
			log.Error(err)
		} else if revision > d.helmReleaseRevision {
			log.Warnf("Detected Helm revision %d higher than current revision %d, stopping sensor", revision, d.helmReleaseRevision)
			d.sensor.Stop()
		}
	}

	protoSecret := getProtoSecret(secret)
	populateTypeData(protoSecret, secret.Data)
	return []*central.SensorEvent{secretToSensorEvent(action, protoSecret)}
}

// isHelmSecret returns whether the secret is used by Helm to store release information.
func isHelmSecret(secret *v1.Secret) bool {
	_, ok := GetHelmSecretTypes()[secret.Type]
	return ok
}

// GetHelmSecretTypes returns a map with each secret type that Helm uses to store
// release information in the keys, and with `true` as value.
func GetHelmSecretTypes() map[v1.SecretType]bool {
	return map[v1.SecretType]bool{
		secretTypeHelmReleaseV1: true,
	}
}

// ExtractHelmRevisionFromHelmSecret Extracts the Helm release revision number from the secret where Helm stores
// the release metadata starting with Helm 3.
// Assuming the following naming conventions:
// - For secretTypeHelmReleaseV1: "sh.helm.release.v1.RELEASE_NAME.vREVISION".
// Returns
// - 0, nil if the secret corresponds to a release different to `helmReleaseName`.
// - 0, err if the secret is not a helm secret (see isHelmSecret), or the secret name doesn't have the expected format.
// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
func ExtractHelmRevisionFromHelmSecret(helmReleaseName string, secret *v1.Secret) (uint64, error) {
	if secret.Type == secretTypeHelmReleaseV1 {
		secretName := secret.Name
		splitSecretName := strings.Split(secretName, ".")
		if len(splitSecretName) != 6 {
			return 0, errors.Errorf("unexpected format for Helm release revision %s", secretName)
		}
		if splitSecretName[4] != helmReleaseName {
			return 0, nil
		}
		rev, err := strconv.Atoi(splitSecretName[5][1:])
		if err != nil {
			return 0, errors.Wrapf(err, "unexpected format for Helm release revision %s, revision is not an int", secretName)
		}
		if rev <= 0 {
			return 0, errors.Errorf("unexpected format for Helm release revision %s, revision is not a positive int", secretName)
		}
		return uint64(rev), nil
	}
	return 0, errors.Errorf("unexpected type %s for secret with name %s", secret.Type, secret.Name)
}