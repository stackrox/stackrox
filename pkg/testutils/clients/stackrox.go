package clients

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/testutils/credentials"
	"google.golang.org/grpc"
)

// StackRoxClients provides a unified interface to all StackRox API services
type StackRoxClients struct {
	conn *grpc.ClientConn

	// Core Services
	Alerts     AlertClient
	Policies   PolicyClient
	Images     ImageClient
	Clusters   ClusterClient
	Auth       AuthClient
	Backups    BackupClient

	// Management Services
	APITokens      APITokenClient
	Roles          RoleClient
	Config         ConfigClient
	Features       FeatureClient

	// Compliance Services
	Compliance     ComplianceClient
	NetworkPolicy  NetworkPolicyClient

	// Integration Services
	Integrations   IntegrationClient
}

// NewStackRoxClients creates a new StackRox client collection using centralized credentials
func NewStackRoxClients(t testutils.T, creds *credentials.Credentials) *StackRoxClients {
	conn := centralgrpc.GRPCConnectionToCentral(t)

	return &StackRoxClients{
		conn: conn,

		// Initialize all service clients
		Alerts:        &alertClient{conn: conn},
		Policies:      &policyClient{conn: conn},
		Images:        &imageClient{conn: conn},
		Clusters:      &clusterClient{conn: conn},
		Auth:          &authClient{conn: conn},
		Backups:       &backupClient{conn: conn},
		APITokens:     &apiTokenClient{conn: conn},
		Roles:         &roleClient{conn: conn},
		Config:        &configClient{conn: conn},
		Features:      &featureClient{conn: conn},
		Compliance:    &complianceClient{conn: conn},
		NetworkPolicy: &networkPolicyClient{conn: conn},
		Integrations:  &integrationClient{conn: conn},
	}
}

// Close closes the underlying gRPC connection
func (c *StackRoxClients) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// PolicyClient provides access to policy management APIs
type PolicyClient interface {
	CreatePolicy(ctx context.Context, policy *PolicyConfig) (*storage.Policy, error)
	GetPolicy(ctx context.Context, policyID string) (*storage.Policy, error)
	UpdatePolicy(ctx context.Context, policy *storage.Policy) (*storage.Policy, error)
	DeletePolicy(ctx context.Context, policyID string) error
	ListPolicies(ctx context.Context) ([]*storage.ListPolicy, error)
	ImportPolicies(ctx context.Context, policies []*storage.Policy) error
}

// PolicyConfig represents the configuration for creating a test policy
type PolicyConfig struct {
	Name                string
	Description         string
	Categories          []string
	Enforcement         bool
	Scope               PolicyScope
	Severity            storage.Severity
	NetworkPolicyConfig *NetworkPolicyConfig
	RuntimeConfig       *RuntimePolicyConfig
	BuildConfig         *BuildPolicyConfig
}

type PolicyScope string

const (
	RuntimeScope PolicyScope = "runtime"
	BuildScope   PolicyScope = "build"
)

type NetworkPolicyConfig struct {
	BlockExternalEgress bool
	AllowedPorts        []int32
	RequiredLabels      map[string]string
}

type RuntimePolicyConfig struct {
	RequireNonRoot      bool
	BlockPrivileged     bool
	ReadOnlyRootFS      bool
	DisallowedCommands  []string
}

type BuildPolicyConfig struct {
	RequiredLabels      map[string]string
	DisallowedPackages  []string
	MaxCVSSScore        float64
	BlockedRegistries   []string
}

// AlertClient provides access to alert APIs
type AlertClient interface {
	GetAlert(ctx context.Context, alertID string) (*storage.Alert, error)
	ListAlerts(ctx context.Context, query string) ([]*storage.ListAlert, error)
	GetAlertsForPolicy(ctx context.Context, policyID string) ([]*storage.Alert, error)
	GetAlertsForDeployment(ctx context.Context, deploymentID string) ([]*storage.Alert, error)
	GetAlertsForImage(ctx context.Context, imageName string) ([]*storage.Alert, error)
	ResolveAlert(ctx context.Context, alertID string) error
	DeleteAlert(ctx context.Context, alertID string) error
}

// ImageClient provides access to image scanning APIs
type ImageClient interface {
	ScanImage(ctx context.Context, imageName string) (*ImageScanResult, error)
	GetImage(ctx context.Context, imageID string) (*storage.Image, error)
	ListImages(ctx context.Context, query string) ([]*storage.ListImage, error)
	GetImageVulnerabilities(ctx context.Context, imageID string) ([]*storage.ImageVulnerability, error)
}

type ImageScanResult struct {
	Image           string
	Vulnerabilities []*storage.ImageVulnerability
	ScanTime        time.Time
	Status          string
	Components      []*storage.ImageComponent
}

// ClusterClient provides access to cluster management APIs
type ClusterClient interface {
	GetCluster(ctx context.Context, clusterID string) (*storage.Cluster, error)
	ListClusters(ctx context.Context) ([]*storage.Cluster, error)
	GetClusterStatus(ctx context.Context, clusterID string) (*storage.ClusterStatus, error)
}

// AuthClient provides access to authentication APIs
type AuthClient interface {
	WhoAmI(ctx context.Context) (*storage.AuthStatus, error)
	GetAuthProviders(ctx context.Context) ([]*storage.AuthProvider, error)
}

// BackupClient provides access to backup management APIs
type BackupClient interface {
	CreateBackup(ctx context.Context, config *BackupConfig) (*storage.ExternalBackup, error)
	TriggerBackup(ctx context.Context, backupID string) error
	GetBackupStatus(ctx context.Context, backupID string) (*storage.ExternalBackup, error)
	ListBackups(ctx context.Context) ([]*storage.ExternalBackup, error)
	DeleteBackup(ctx context.Context, backupID string) error
}

type BackupConfig struct {
	Name            string
	Schedule        string
	IncludeCerts    bool
	IncludeKeys     bool
	BackupLocation  string
	S3Config        *S3BackupConfig
	GCSConfig       *GCSBackupConfig
	AzureConfig     *AzureBackupConfig
}

type S3BackupConfig struct {
	Bucket     string
	Region     string
	AccessKey  string
	SecretKey  string
}

type GCSBackupConfig struct {
	Bucket            string
	ServiceAccount    string
}

type AzureBackupConfig struct {
	Container         string
	StorageAccount    string
	AccountKey        string
}

// APITokenClient provides access to API token management
type APITokenClient interface {
	CreateToken(ctx context.Context, name string, roles []string) (*TokenResponse, error)
	GetTokens(ctx context.Context) ([]*storage.TokenMetadata, error)
	RevokeToken(ctx context.Context, tokenID string) error
}

type TokenResponse struct {
	ID    string
	Token string
}

// RoleClient provides access to role and permission management
type RoleClient interface {
	CreateRole(ctx context.Context, role *storage.Role) (*storage.Role, error)
	GetRoles(ctx context.Context) ([]*storage.Role, error)
	DeleteRole(ctx context.Context, roleID string) error
	CreatePermissionSet(ctx context.Context, permSet *storage.PermissionSet) (*storage.PermissionSet, error)
	GetPermissionSets(ctx context.Context) ([]*storage.PermissionSet, error)
	DeletePermissionSet(ctx context.Context, permSetID string) error
}

// ConfigClient provides access to system configuration
type ConfigClient interface {
	GetConfig(ctx context.Context) (*storage.Config, error)
	UpdateConfig(ctx context.Context, config *storage.Config) (*storage.Config, error)
	GetPublicConfig(ctx context.Context) (*storage.PublicConfig, error)
}

// FeatureClient provides access to feature flag management
type FeatureClient interface {
	GetFeatureFlags(ctx context.Context) (map[string]bool, error)
	SetFeatureFlag(ctx context.Context, flag string, enabled bool) error
}

// ComplianceClient provides access to compliance management
type ComplianceClient interface {
	GetComplianceResults(ctx context.Context, query string) (*storage.ComplianceResults, error)
	TriggerComplianceScan(ctx context.Context, clusterID string) error
	GetComplianceStandards(ctx context.Context) ([]*storage.ComplianceStandard, error)
}

// NetworkPolicyClient provides access to network policy management
type NetworkPolicyClient interface {
	GetNetworkPolicies(ctx context.Context, query string) ([]*storage.NetworkPolicy, error)
	ApplyNetworkPolicy(ctx context.Context, policy *storage.NetworkPolicy) error
	DeleteNetworkPolicy(ctx context.Context, policyID string) error
	SimulateNetworkPolicy(ctx context.Context, policy *storage.NetworkPolicy) (*NetworkPolicySimulation, error)
}

type NetworkPolicySimulation struct {
	AllowedConnections []string
	BlockedConnections []string
	Violations         []string
}

// IntegrationClient provides access to external integrations
type IntegrationClient interface {
	GetNotifiers(ctx context.Context) ([]*storage.Notifier, error)
	CreateNotifier(ctx context.Context, notifier *storage.Notifier) (*storage.Notifier, error)
	TestNotifier(ctx context.Context, notifierID string) error
	DeleteNotifier(ctx context.Context, notifierID string) error
	GetImageIntegrations(ctx context.Context) ([]*storage.ImageIntegration, error)
	CreateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) (*storage.ImageIntegration, error)
	TestImageIntegration(ctx context.Context, integrationID string) error
}

// Common error types
var (
	ErrNotFound         = fmt.Errorf("resource not found")
	ErrAlreadyExists    = fmt.Errorf("resource already exists")
	ErrInvalidConfig    = fmt.Errorf("invalid configuration")
	ErrUnauthorized     = fmt.Errorf("unauthorized")
	ErrServiceUnavailable = fmt.Errorf("service unavailable")
)