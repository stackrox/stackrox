/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// Important: Run "make generate manifests" to regenerate code after modifying this file

// -------------------------------------------------------------
// Spec

// CentralSpec defines the desired state of Central
type CentralSpec struct {
	// Settings for the Central component, which is responsible for all user interaction.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Central Component Settings"
	Central *CentralComponentSpec `json:"central,omitempty"`

	// Settings for the Scanner component, which is responsible for vulnerability scanning of container
	// images.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,displayName="Scanner Component Settings"
	Scanner *ScannerComponentSpec `json:"scanner,omitempty"`

	// Settings related to outgoing network traffic.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	Egress *Egress `json:"egress,omitempty"`

	// Allows you to specify additional trusted Root CAs.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	TLS *TLSConfig `json:"tls,omitempty"`

	// Additional image pull secrets to be taken into account for pulling images.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image Pull Secrets",order=5,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	ImagePullSecrets []LocalSecretReference `json:"imagePullSecrets,omitempty"`

	// Customizations to apply on all Central Services components.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Customizations,order=6,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Customize *CustomizeSpec `json:"customize,omitempty"`

	// Miscellaneous settings.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Miscellaneous,order=7,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Misc *MiscSpec `json:"misc,omitempty"`

	// Overlays
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Overlays,order=8,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Overlays []*K8sObjectOverlay `json:"overlays,omitempty"`

	// Monitoring configuration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced"}
	Monitoring *GlobalMonitoring `json:"monitoring,omitempty"`
}

// Egress defines settings related to outgoing network traffic.
type Egress struct {
	// Configures whether Red Hat Advanced Cluster Security should run in online or offline (disconnected) mode.
	// In offline mode, automatic updates of vulnerability definitions and kernel modules are disabled.
	//+kubebuilder:default=Online
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName=Connectivity Policy,order=1
	ConnectivityPolicy *ConnectivityPolicy `json:"connectivityPolicy,omitempty"`
}

// ConnectivityPolicy is a type for values of spec.egress.connectivityPolicy.
// +kubebuilder:validation:Enum=Online;Offline
type ConnectivityPolicy string

const (
	// ConnectivityOnline means that Central is allowed to make outbound connections to the Internet.
	ConnectivityOnline ConnectivityPolicy = "Online"
	// ConnectivityOffline means that Central must not make outbound connections to the Internet.
	ConnectivityOffline ConnectivityPolicy = "Offline"
)

// CentralComponentSpec defines settings for the "central" component.
type CentralComponentSpec struct {
	// Specify a secret that contains the administrator password in the "password" data item.
	// If omitted, the operator will auto-generate a password and store it in the "password" item
	// in the "central-htpasswd" secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Administrator Password",order=1
	AdminPasswordSecret *LocalSecretReference `json:"adminPasswordSecret,omitempty"`

	// Disable admin password generation. Do not use this for first-time installations,
	// as you will have no way to perform initial setup and configuration of alternative authentication methods.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	AdminPasswordGenerationDisabled *bool `json:"adminPasswordGenerationDisabled,omitempty"`

	// Here you can configure if you want to expose central through a node port, a load balancer, or an OpenShift
	// route.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Exposure *Exposure `json:"exposure,omitempty"`

	// By default, Central will only serve an internal TLS certificate, which means that you will
	// need to handle TLS termination at the ingress or load balancer level.
	// If you want to terminate TLS in Central and serve a custom server certificate, you can specify
	// a secret containing the certificate and private key here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="User-facing TLS certificate secret",order=3
	DefaultTLSSecret *LocalSecretReference `json:"defaultTLSSecret,omitempty"`

	// Configures monitoring endpoint for Central. The monitoring endpoint
	// allows other services to collect metrics from Central, provided in
	// Prometheus compatible format.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	Monitoring *Monitoring `json:"monitoring,omitempty"`

	// Configures how Central should store its persistent data. You can choose between using a persistent
	// volume claim (recommended default), and a host path.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=5
	Persistence *Persistence `json:"persistence,omitempty"`

	// Settings for Central DB, which is responsible for data persistence.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=6,displayName="Central DB Settings"
	DB *CentralDBSpec `json:"db,omitempty"`

	// Configures telemetry settings for Central. If enabled, Central transmits telemetry and diagnostic
	// data to a remote storage backend.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=7,displayName="Telemetry",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	Telemetry *Telemetry `json:"telemetry,omitempty"`

	// Configures resources within Central in a declarative manner.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=8,displayName="Declarative Configuration",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	DeclarativeConfiguration *DeclarativeConfiguration `json:"declarativeConfiguration,omitempty"`

	// Configures the encryption of notifier secrets stored in the Central DB.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=9,displayName="Notifier Secrets Encryption",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	NotifierSecretsEncryption *NotifierSecretsEncryption `json:"notifierSecretsEncryption,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=99
	DeploymentSpec `json:",inline"`
}

// GetDB returns Central's db config
func (c *CentralComponentSpec) GetDB() *CentralDBSpec {
	if c == nil {
		return nil
	}
	return c.DB
}

// GetPersistence returns Central's persistence config
func (c *CentralComponentSpec) GetPersistence() *Persistence {
	if c == nil {
		return nil
	}
	return c.Persistence
}

// GetAdminPasswordSecret provides a way to retrieve the admin password that is safe to use on a nil receiver object.
func (c *CentralComponentSpec) GetAdminPasswordSecret() *LocalSecretReference {
	if c == nil {
		return nil
	}
	return c.AdminPasswordSecret
}

// GetAdminPasswordGenerationDisabled provides a way to retrieve the AdminPasswordEnabled setting that is safe to use on a nil receiver object.
func (c *CentralComponentSpec) GetAdminPasswordGenerationDisabled() bool {
	if c == nil {
		return false
	}
	return pointer.BoolDeref(c.AdminPasswordGenerationDisabled, false)
}

// IsExternalDB returns true if central DB is not managed by the Operator
func (c *CentralComponentSpec) IsExternalDB() bool {
	return c != nil && c.DB.IsExternal()
}

// GetNotifierSecretsEncryptionEnabled provides a way to retrieve the NotifierSecretsEncryption.Enabled setting that is safe to use on a nil receiver object.
func (c *CentralComponentSpec) GetNotifierSecretsEncryptionEnabled() bool {
	if c == nil || c.NotifierSecretsEncryption == nil {
		return false
	}
	return pointer.BoolDeref(c.NotifierSecretsEncryption.Enabled, false)
}

// DeclarativeConfiguration defines settings for adding resources in a declarative manner.
type DeclarativeConfiguration struct {
	// List of config maps containing declarative configuration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Config maps containing declarative configuration"
	ConfigMaps []LocalConfigMapReference `json:"configMaps,omitempty"`

	// List of secrets containing declarative configuration.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secrets containing declarative configuration"
	Secrets []LocalSecretReference `json:"secrets,omitempty"`
}

// NotifierSecretsEncryption defines settings for encrypting notifier secrets in the Central DB.
type NotifierSecretsEncryption struct {
	// Enables the encryption of notifier secrets stored in the Central DB. An encryption key must be
	// provided in a secret called `central-encryption-key` in the Central namespace, with the key stored in
	// the `encryption-key` data field.
	//+kubebuilder:default=false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Enabled *bool `json:"enabled,omitempty"`
}

// CentralDBSpec defines settings for the "central db" component.
// TODO(ROX-14395): drop `IsEnabled` field when bumping API version.
// isEnabled is effectively no-op starting from the version 3.74.0. It should be removed when we
// bump API version of ACS custom resources. Removing it before is unsafe and may break compatibility.
type CentralDBSpec struct {
	// Deprecated field. It is no longer necessary to specify it.
	// This field will be removed in a future release.
	// Central is configured to use PostgreSQL by default.
	//+kubebuilder:default=Default
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:hidden"}
	IsEnabled *CentralDBEnabled `json:"isEnabled,omitempty"`

	// Specify a secret that contains the password in the "password" data item. This can only be used when
	// specifying a connection string manually.
	// When omitted, the operator will auto-generate a DB password and store it in the "password" item
	// in the "central-db-password" secret.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Administrator Password",order=1
	PasswordSecret *LocalSecretReference `json:"passwordSecret,omitempty"`

	// Specify a connection string that corresponds to an external database. If set, the operator will not manage Central DB.
	// When using this option, you must explicitly set a password secret; automatically generating a password will not
	// be supported.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,displayName="Connection String"
	ConnectionStringOverride *string `json:"connectionString,omitempty"`

	// Configures how Central DB should store its persistent data. You can choose between using a persistent
	// volume claim (recommended default), and a host path.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	Persistence *DBPersistence `json:"persistence,omitempty"`

	// Config map containing postgresql.conf and pg_hba.conf that will be used if modifications need to be applied.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Config map that will override postgresql.conf and pg_hba.conf"
	ConfigOverride LocalConfigMapReference `json:"configOverride,omitempty"`

	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=99
	DeploymentSpec `json:",inline"`
}

// CentralDBEnabled is a type for values of spec.central.db.enabled.
// +kubebuilder:validation:Enum=Default;Enabled
type CentralDBEnabled string

const (
	// CentralDBEnabledDefault configures the central to use PostgreSQL database.
	// Deprecated const.
	CentralDBEnabledDefault CentralDBEnabled = "Default"

	// CentralDBEnabledTrue configures the central to use a PostgreSQL database.
	// Deprecated const.
	CentralDBEnabledTrue CentralDBEnabled = "Enabled"
)

// CentralDBEnabledPtr return a pointer for the given CentralDBEnabled value
func CentralDBEnabledPtr(c CentralDBEnabled) *CentralDBEnabled {
	ptr := new(CentralDBEnabled)
	*ptr = c
	return ptr
}

// GetPasswordSecret provides a way to retrieve the admin password that is safe to use on a nil receiver object.
func (c *CentralDBSpec) GetPasswordSecret() *LocalSecretReference {
	if c == nil {
		return nil
	}
	return c.PasswordSecret
}

// IsExternal specifies that the database should not be managed by the Operator
func (c *CentralDBSpec) IsExternal() bool {
	if c == nil {
		return false
	}
	return c.ConnectionStringOverride != nil
}

// GetPersistence returns the persistence for Central DB
func (c *CentralDBSpec) GetPersistence() *DBPersistence {
	if c == nil {
		return nil
	}
	return c.Persistence
}

// Persistence defines persistence settings for central.
type Persistence struct {
	// Uses a Kubernetes persistent volume claim (PVC) to manage the storage location of persistent data.
	// Recommended for most users.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Persistent volume claim",order=1
	PersistentVolumeClaim *PersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`

	// Stores persistent data on a directory on the host. This is not recommended, and should only
	// be used together with a node selector (only available in YAML view).
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Host path",order=99
	HostPath *HostPathSpec `json:"hostPath,omitempty"`
}

// GetPersistentVolumeClaim returns the configured PVC
func (p *Persistence) GetPersistentVolumeClaim() *PersistentVolumeClaim {
	if p == nil {
		return nil
	}
	return p.PersistentVolumeClaim
}

// GetHostPath returns the configured host path
func (p *Persistence) GetHostPath() string {
	if p == nil {
		return ""
	}
	if p.HostPath == nil {
		return ""
	}

	return pointer.StringDeref(p.HostPath.Path, "")
}

// HostPathSpec defines settings for host path config.
type HostPathSpec struct {
	// The path on the host running Central.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=99
	Path *string `json:"path,omitempty"`
}

// PersistentVolumeClaim defines PVC-based persistence settings.
type PersistentVolumeClaim struct {
	// The name of the PVC to manage persistent data. If no PVC with the given name exists, it will be
	// created. Defaults to "stackrox-db" if not set.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Claim Name",order=1
	//+kubebuilder:default=stackrox-db
	ClaimName *string `json:"claimName,omitempty"`

	// The size of the persistent volume when created through the claim. If a claim was automatically created,
	// this can be used after the initial deployment to resize (grow) the volume (only supported by some
	// storage class controllers).
	//+kubebuilder:validation:Pattern=^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Size",order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Size *string `json:"size,omitempty"`

	// The name of the storage class to use for the PVC. If your cluster is not configured with a default storage
	// class, you must select a value here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage Class",order=3,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass"}
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// DBPersistence defines persistence settings for Central DB.
type DBPersistence struct {
	// Uses a Kubernetes persistent volume claim (PVC) to manage the storage location of persistent data.
	// Recommended for most users.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Persistent volume claim",order=1
	PersistentVolumeClaim *DBPersistentVolumeClaim `json:"persistentVolumeClaim,omitempty"`

	// Stores persistent data on a directory on the host. This is not recommended, and should only
	// be used together with a node selector (only available in YAML view).
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Host path",order=99
	HostPath *HostPathSpec `json:"hostPath,omitempty"`
}

// GetPersistentVolumeClaim returns the configured PVC
func (p *DBPersistence) GetPersistentVolumeClaim() *DBPersistentVolumeClaim {
	if p == nil {
		return nil
	}
	return p.PersistentVolumeClaim
}

// GetHostPath returns the configured host path
func (p *DBPersistence) GetHostPath() string {
	if p == nil {
		return ""
	}
	if p.HostPath == nil {
		return ""
	}

	return pointer.StringDeref(p.HostPath.Path, "")
}

// DBPersistentVolumeClaim defines PVC-based persistence settings for Central DB.
type DBPersistentVolumeClaim struct {
	// The name of the PVC to manage persistent data. If no PVC with the given name exists, it will be
	// created. Defaults to "central-db" if not set.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Claim Name",order=1
	//+kubebuilder:default=central-db
	ClaimName *string `json:"claimName,omitempty"`

	// The size of the persistent volume when created through the claim. If a claim was automatically created,
	// this can be used after the initial deployment to resize (grow) the volume (only supported by some
	// storage class controllers).
	//+kubebuilder:validation:Pattern=^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Size",order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Size *string `json:"size,omitempty"`

	// The name of the storage class to use for the PVC. If your cluster is not configured with a default storage
	// class, you must select a value here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Storage Class",order=3,xDescriptors={"urn:alm:descriptor:io.kubernetes:StorageClass"}
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// Exposure defines how central is exposed.
type Exposure struct {
	// Expose Central through an OpenShift route.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Route"
	Route *ExposureRoute `json:"route,omitempty"`

	// Expose Central through a load balancer service.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,displayName="Load Balancer"
	LoadBalancer *ExposureLoadBalancer `json:"loadBalancer,omitempty"`

	// Expose Central through a node port.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="Node Port"
	NodePort *ExposureNodePort `json:"nodePort,omitempty"`
}

// ExposureLoadBalancer defines settings for exposing central via a LoadBalancer.
type ExposureLoadBalancer struct {
	//+kubebuilder:default=false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Enabled *bool `json:"enabled,omitempty"`

	// Defaults to 443 if not set.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=65535
	//+kubebuilder:default=443
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true"}
	Port *int32 `json:"port,omitempty"`

	// If you have a static IP address reserved for your load balancer, you can enter it here.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true"}
	IP *string `json:"ip,omitempty"`
}

// ExposureNodePort defines settings for exposing central via a NodePort.
type ExposureNodePort struct {
	//+kubebuilder:default=false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Enabled *bool `json:"enabled,omitempty"`

	// Use this to specify an explicit node port. Most users should leave this empty.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=65535
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true"}
	Port *int32 `json:"port,omitempty"`
}

// ExposureRoute defines settings for exposing central via a Route.
type ExposureRoute struct {
	//+kubebuilder:default=false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Enabled *bool `json:"enabled,omitempty"`

	// Specify a custom hostname for the central route.
	// If unspecified, an appropriate default value will be automatically chosen by OpenShift route operator.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Host *string `json:"host,omitempty"`
}

// Telemetry defines telemetry settings for Central.
type Telemetry struct {
	// Specifies if Telemetry is enabled.
	//+kubebuilder:validation:Default=true
	//+kubebuilder:default=true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	Enabled *bool `json:"enabled,omitempty"`

	// Defines the telemetry storage backend for Central.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true"}
	Storage *TelemetryStorage `json:"storage,omitempty"`
}

// TelemetryStorage defines the telemetry storage backend for Central.
type TelemetryStorage struct {
	// Storage API endpoint.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Endpoint *string `json:"endpoint,omitempty"`

	// Storage API key. If not set, telemetry is disabled.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Key *string `json:"key,omitempty"`
}

// Note the following struct should mostly match LocalScannerComponentSpec for the SecuredCluster type. Different Scanner
// types struct are maintained because of UI exposed documentation differences.

// ScannerComponentSpec defines settings for the central "scanner" component.
type ScannerComponentSpec struct {
	// If you do not want to deploy the Red Hat Advanced Cluster Security Scanner, you can disable it here
	// (not recommended). By default, the scanner is enabled.
	// If you do so, all the settings in this section will have no effect.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Scanner Component",order=1
	ScannerComponent *ScannerComponentPolicy `json:"scannerComponent,omitempty"`

	// Settings pertaining to the analyzer deployment, such as for autoscaling.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.scannerComponent:Enabled"}
	Analyzer *ScannerAnalyzerComponent `json:"analyzer,omitempty"`

	// Settings pertaining to the database used by the Red Hat Advanced Cluster Security Scanner.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="DB",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.scannerComponent:Enabled"}
	DB *DeploymentSpec `json:"db,omitempty"`

	// Configures monitoring endpoint for Scanner. The monitoring endpoint
	// allows other services to collect metrics from Scanner, provided in
	// Prometheus compatible format.
	//+operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	Monitoring *Monitoring `json:"monitoring,omitempty"`
}

// GetAnalyzer returns the analyzer component even if receiver is nil
func (s *ScannerComponentSpec) GetAnalyzer() *ScannerAnalyzerComponent {
	if s == nil {
		return nil
	}
	return s.Analyzer
}

// IsEnabled checks whether scanner is enabled. This method is safe to be used with nil receivers.
func (s *ScannerComponentSpec) IsEnabled() bool {
	if s == nil || s.ScannerComponent == nil {
		return true // enabled by default
	}
	return *s.ScannerComponent == ScannerComponentEnabled
}

// ScannerComponentPolicy is a type for values of spec.scanner.scannerComponent.
// +kubebuilder:validation:Enum=Enabled;Disabled
type ScannerComponentPolicy string

const (
	// ScannerComponentEnabled means that scanner should be installed.
	ScannerComponentEnabled ScannerComponentPolicy = "Enabled"
	// ScannerComponentDisabled means that scanner should not be installed.
	ScannerComponentDisabled ScannerComponentPolicy = "Disabled"
)

// -------------------------------------------------------------
// Status

// CentralStatus defines the observed state of Central.
type CentralStatus struct {
	Conditions      []StackRoxCondition `json:"conditions"`
	DeployedRelease *StackRoxRelease    `json:"deployedRelease,omitempty"`

	// The deployed version of the product.
	//+operator-sdk:csv:customresourcedefinitions:type=status,order=1
	ProductVersion string `json:"productVersion,omitempty"`
	//+operator-sdk:csv:customresourcedefinitions:type=status,order=2
	Central *CentralComponentStatus `json:"central,omitempty"`
}

// AdminPasswordStatus shows status related to the admin password.
type AdminPasswordStatus struct {
	// Info stores information on how to obtain the admin password.
	//+operator-sdk:csv:customresourcedefinitions:type=status,order=1,displayName="Admin Credentials Info"
	Info string `json:"info,omitempty"`

	// AdminPasswordSecretReference contains reference for the admin password
	//+operator-sdk:csv:customresourcedefinitions:type=status,order=2,displayName="Admin Password Secret Reference",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	SecretReference *string `json:"adminPasswordSecretReference,omitempty"`
}

// CentralComponentStatus describes status specific to the central component.
type CentralComponentStatus struct {
	// AdminPassword stores information related to the auto-generated admin password.
	AdminPassword *AdminPasswordStatus `json:"adminPassword,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+operator-sdk:csv:customresourcedefinitions:resources={{Deployment,v1,""},{Secret,v1,""},{Service,v1,""},{Route,v1,""}}
//+genclient

// Central is the configuration template for the central services. This includes the API server, persistent storage,
// and the web UI, as well as the image scanner.
type Central struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CentralSpec   `json:"spec,omitempty"`
	Status CentralStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CentralList contains a list of Central
type CentralList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Central `json:"items"`
}

var (
	// CentralGVK is the GVK for the Central type.
	CentralGVK = SchemeGroupVersion.WithKind("Central")
)

// IsScannerEnabled returns true if scanner is enabled.
func (c *Central) IsScannerEnabled() bool {
	return c.Spec.Scanner.IsEnabled()
}
