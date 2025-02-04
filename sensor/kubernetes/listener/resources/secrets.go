package resources

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudflare/cfssl/certinfo"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/openshift"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/managedcentral"
	"github.com/stackrox/rox/sensor/common/registry"
	"github.com/stackrox/rox/sensor/common/store/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/core/v1"
)

const (
	// clusterImgRegistryOperatorNamespace is the namespace where the cluster image registry
	// operator runs.
	clusterImgRegistryOperatorNamespace = "openshift-image-registry"

	// clusterImgRegistryOperatorSecretName is the name of the secret used by the cluster image registry
	// operator that is a known copy of the OCP global pull secret.
	clusterImgRegistryOperatorSecretName = "installation-pull-secrets"
)

var (
	ocpSAAnnotations = []string{
		"kubernetes.io/service-account.name",
		"openshift.io/internal-registry-auth-token.service-account",
	}

	dataTypeMap = map[string]storage.SecretType{
		"-----BEGIN CERTIFICATE-----":              storage.SecretType_PUBLIC_CERTIFICATE,
		"-----BEGIN NEW CERTIFICATE REQUEST-----":  storage.SecretType_CERTIFICATE_REQUEST,
		"-----BEGIN PRIVACY-ENHANCED MESSAGE-----": storage.SecretType_PRIVACY_ENHANCED_MESSAGE,
		"-----BEGIN OPENSSH PRIVATE KEY-----":      storage.SecretType_OPENSSH_PRIVATE_KEY,   // notsecret
		"-----BEGIN PGP PRIVATE KEY BLOCK-----":    storage.SecretType_PGP_PRIVATE_KEY,       // notsecret
		"-----BEGIN EC PRIVATE KEY-----":           storage.SecretType_EC_PRIVATE_KEY,        // notsecret
		"-----BEGIN RSA PRIVATE KEY-----":          storage.SecretType_RSA_PRIVATE_KEY,       // notsecret
		"-----BEGIN DSA PRIVATE KEY-----":          storage.SecretType_DSA_PRIVATE_KEY,       // notsecret
		"-----BEGIN PRIVATE KEY-----":              storage.SecretType_CERT_PRIVATE_KEY,      // notsecret
		"-----BEGIN ENCRYPTED PRIVATE KEY-----":    storage.SecretType_ENCRYPTED_PRIVATE_KEY, // notsecret
	}
)

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
type secretDispatcher struct {
	regStore *registry.Store
}

// newSecretDispatcher creates and returns a new secret handler.
func newSecretDispatcher(regStore *registry.Store) *secretDispatcher {
	return &secretDispatcher{
		regStore: regStore,
	}
}

func deriveIDFromSecret(secret *v1.Secret, registry string) (string, error) {
	rootUUID, err := uuid.FromString(string(secret.UID))
	if err != nil {
		return "", errors.Wrapf(err, "converting secret ID %q to uuid", secret.UID)
	}
	id := uuid.NewV5(rootUUID, registry).String()
	return id, nil
}

// DockerConfigToImageIntegration creates an image integration for a given
// registry URL and docker config.
func DockerConfigToImageIntegration(secret *v1.Secret, registry string, dce config.DockerConfigEntry) (*storage.ImageIntegration, error) {
	registryType := types.DockerType
	if rhel.RedHatRegistryEndpoints.Contains(urlfmt.TrimHTTPPrefixes(registry)) {
		registryType = types.RedHatType
	}

	sourcedIntegration := shouldCreateSourcedIntegration(secret)

	var id string
	if !sourcedIntegration {
		id = uuid.NewV4().String()
	} else {
		var err error
		id, err = deriveIDFromSecret(secret, registry)
		if err != nil {
			return nil, errors.Wrapf(err, "deriving image integration ID from secret %q", secret.UID)
		}
	}
	ii := &storage.ImageIntegration{
		Id:         id,
		Type:       registryType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Username: dce.Username,
				Password: dce.Password,
			},
		},
		Autogenerated: true,
	}

	if sourcedIntegration {
		ii.Source = &storage.ImageIntegration_Source{
			ClusterId:           clusterid.Get(),
			Namespace:           secret.GetNamespace(),
			ImagePullSecretName: secret.GetName(),
		}
	}

	return ii, nil
}

func getDockerConfigFromSecret(secret *v1.Secret) config.DockerConfig {
	var dockerConfig config.DockerConfig
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
		var dockerConfigJSON config.DockerConfigJSON
		if err := json.Unmarshal(data, &dockerConfigJSON); err != nil {
			log.Error(err)
			return nil
		}
		dockerConfig = dockerConfigJSON.Auths
	default:
		utils.Should(errors.New("only Docker Config secrets are allowed"))
		return nil
	}
	return dockerConfig
}

func imageIntegationIDSetFromSecret(secret *v1.Secret) (set.StringSet, error) {
	if secret == nil {
		return nil, nil
	}
	dockerConfig := getDockerConfigFromSecret(secret)
	if len(dockerConfig) == 0 {
		return nil, nil
	}
	imageIntegrationIDSet := set.NewStringSet()
	for reg := range dockerConfig {
		id, err := deriveIDFromSecret(secret, reg)
		if err != nil {
			return nil, err
		}
		imageIntegrationIDSet.Add(id)
	}
	return imageIntegrationIDSet, nil
}

// shouldCreateSourcedIntegration will return true if integrations created from this secret
// should include source details.
func shouldCreateSourcedIntegration(secret *v1.Secret) bool {
	return features.SourcedAutogeneratedIntegrations.Enabled() ||
		(env.AutogenerateGlobalPullSecRegistries.BooleanSetting() && openshift.GlobalPullSecret(secret.GetNamespace(), secret.GetName()))
}

func (s *secretDispatcher) processDockerConfigEvent(secret, oldSecret *v1.Secret, action central.ResourceAction) *component.ResourceEvent {
	dockerConfig := getDockerConfigFromSecret(secret)
	if len(dockerConfig) == 0 {
		return nil
	}

	sensorEvents := make([]*central.SensorEvent, 0, len(dockerConfig)+1)
	registries := make([]*storage.ImagePullSecret_Registry, 0, len(dockerConfig))

	var ocpServiceAccountName string
	for _, annotation := range ocpSAAnnotations {
		if name, ok := secret.GetAnnotations()[annotation]; ok {
			ocpServiceAccountName = name
			break
		}
	}

	s.processSecretForLocalScanning(secret, action, dockerConfig, ocpServiceAccountName)

	// A sourced integration is one which includes the cluster, namespace, and secret name from which it came.
	sourcedIntegration := shouldCreateSourcedIntegration(secret)

	newIntegrationSet := set.NewStringSet()
	for registryAddress, dce := range dockerConfig {
		registryAddr := strings.TrimSpace(registryAddress)
		if registryAddr != registryAddress {
			log.Warnf("Spaces have been trimmed from registry address %q found in secret %s/%s",
				registryAddress, secret.GetNamespace(), secret.GetName())
		}

		registries = append(registries, &storage.ImagePullSecret_Registry{
			Name:     registryAddr,
			Username: dce.Username,
		})

		// Only send integrations to Central that are not bound to a service account and managed by k8s.
		// This will ignore the secrets generated for the OCP default, builder, deployer, etc. service accounts.
		// These pull secrets are used only for accessing the OCP internal registry, which is only accessible
		// from within the Secured Cluster. Central will be unable to use these credentials.
		if ocpServiceAccountName != "" {
			continue
		}

		if managedcentral.IsCentralManaged() {
			// Do not send image integrations to Central if it's managed (ie: Cloud Service)
			continue
		}

		if skipIntegrationCreate(secret) {
			log.Debugf("Skipping create of image integration for secret %q, namespace %q, registry %q", secret.GetName(), secret.GetNamespace(), registryAddr)
			continue
		}

		ii, err := DockerConfigToImageIntegration(secret, registryAddr, dce)
		if err != nil {
			log.Errorf("unable to create docker config for secret %s: %v", secret.GetName(), err)
			continue
		}

		sensorEvents = append(sensorEvents, &central.SensorEvent{
			// Only update is supported at this time for integrations without a source.
			// Set the Id for sourced integrations so that Sensor's deduper (sensor/common/deduper Send())
			// will NOT suppress subsequent secret remove events.
			Id:     utils.IfThenElse(sourcedIntegration, ii.GetId(), ""),
			Action: action,
			Resource: &central.SensorEvent_ImageIntegration{
				ImageIntegration: ii,
			},
		})

		if sourcedIntegration {
			newIntegrationSet.Add(ii.GetId())
		}
	}

	if sourcedIntegration {
		// Compute diff between old and new secret to automatically clean up delete secrets
		oldIntegrations, err := imageIntegationIDSetFromSecret(oldSecret)
		if err != nil {
			log.Errorf("error getting ids from old secret %q: %v", string(oldSecret.UID), err)
		} else {
			for id := range oldIntegrations.Difference(newIntegrationSet) {
				sensorEvents = append(sensorEvents, &central.SensorEvent{
					Id:     id,
					Action: central.ResourceAction_REMOVE_RESOURCE,
					Resource: &central.SensorEvent_ImageIntegration{
						ImageIntegration: &storage.ImageIntegration{
							Id: id,
							// Add a source to the remove event so that Central can determine
							// if this integrations is from the OCP global pull secret if
							// needed.
							Source: &storage.ImageIntegration_Source{
								ClusterId:           clusterid.Get(),
								Namespace:           secret.GetNamespace(),
								ImagePullSecretName: secret.GetName(),
							},
						},
					},
				})
			}
		}
	}
	sort.SliceStable(registries, func(i, j int) bool {
		if registries[i].Name != registries[j].Name {
			return registries[i].GetName() < registries[j].GetName()
		}
		return registries[i].GetUsername() < registries[j].GetUsername()
	})

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
	events := component.NewEvent(sensorEvents...)
	events.AddSensorEvent(secretToSensorEvent(action, protoSecret))

	// When adding new docker config secrets we need to reprocess every deployment in this cluster.
	// This is because the field `NotPullable` could be updated and hence new image scan results will appear.
	events.AddDeploymentReference(resolver.ResolveAllDeployments())

	return events
}

// skipIntegrationCreate returns true if image integrations should NOT be created from this secret.
func skipIntegrationCreate(secret *v1.Secret) bool {
	if features.SourcedAutogeneratedIntegrations.Enabled() {
		// Create integration if sourced autogen enabled.
		return false
	}

	if env.AutogenerateGlobalPullSecRegistries.BooleanSetting() &&
		secret.GetNamespace() == clusterImgRegistryOperatorNamespace && secret.GetName() == clusterImgRegistryOperatorSecretName {
		// This secret is a copy of the OCP global pull secret, managed by the cluster-image-registry-operator,
		// skip to avoid unnecessary dupes.
		// https://github.com/openshift/cluster-image-registry-operator/blob/release-4.20/pkg/resource/pullsecret.go
		return true
	}

	return false
}

// processSecretForLocalScanning processes and stores secrets to be used for delegated/local scanning.
func (s *secretDispatcher) processSecretForLocalScanning(secret *v1.Secret, action central.ResourceAction, dockerConfig config.DockerConfig, saName string) {
	if !env.LocalImageScanningEnabled.BooleanSetting() {
		// If local scanning is disabled, do not capture secrets.
		return
	}

	if action == central.ResourceAction_REMOVE_RESOURCE {
		if s.regStore.DeleteSecret(secret.GetNamespace(), secret.GetName()) {
			log.Debugf("Deleted secret %q from %q namespace in registry store", secret.GetName(), secret.GetNamespace())
		}
		return
	}

	s.regStore.UpsertSecret(secret.GetNamespace(), secret.GetName(), dockerConfig, saName)
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
		Id:     secret.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Secret{
			Secret: secret,
		},
	}
}

// ProcessEvent processes a secret resource event, and returns the sensor events to emit in response.
func (s *secretDispatcher) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) *component.ResourceEvent {
	secret := obj.(*v1.Secret)

	oldSecret, ok := oldObj.(*v1.Secret)
	if !ok {
		oldSecret = nil
	}

	switch secret.Type {
	case v1.SecretTypeDockerConfigJson, v1.SecretTypeDockercfg:
		return s.processDockerConfigEvent(secret, oldSecret, action)
	case v1.SecretTypeServiceAccountToken:
		// Filter out service account tokens because we have a service account processor.
		return nil
	}

	protoSecret := getProtoSecret(secret)
	populateTypeData(protoSecret, secret.Data)
	return component.NewEvent(secretToSensorEvent(action, protoSecret))
}
