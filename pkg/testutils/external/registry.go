package external

import (
	"fmt"
	"time"

	testenv "github.com/stackrox/rox/pkg/testutils/env"
)

// RegistryClient interface for container registry operations
type RegistryClient interface {
	AuthenticateRegistry() error
	ScanImage(image string) (*ScanResult, error)
	ListRepositories() ([]string, error)
	TestConnection() error
	GetRegistryType() RegistryType
}

type RegistryType string

const (
	GCRRegistry    RegistryType = "gcr"
	ECRRegistry    RegistryType = "ecr"
	ACRRegistry    RegistryType = "acr"
	QuayRegistry   RegistryType = "quay"
	RedHatRegistry RegistryType = "redhat"
	MockRegistry   RegistryType = "mock"
)

// ScanResult represents the result of an image vulnerability scan
type ScanResult struct {
	Image           string
	Vulnerabilities []Vulnerability
	ScanTime        time.Time
	Status          string
	Components      []Component
}

type Vulnerability struct {
	CVE         string
	Severity    string
	CVSS        float64
	Description string
	Package     string
	Version     string
	FixedBy     string
}

type Component struct {
	Name    string
	Version string
	Type    string
}

// GCR Client
type GCRClient struct {
	credentials string
	mock        bool
	mockClient  *mockRegistryClient
}

func NewGCRClient() (*GCRClient, error) {
	credentials := testenv.GoogleGCRCredentials.Setting()

	if credentials == "" || testenv.ShouldUseMockServices() {
		return &GCRClient{
			mock:       true,
			mockClient: newMockRegistryClient(GCRRegistry),
		}, nil
	}

	return &GCRClient{credentials: credentials, mock: false}, nil
}

func (g *GCRClient) AuthenticateRegistry() error {
	if g.mock {
		return g.mockClient.AuthenticateRegistry()
	}
	// TODO: Implement real GCR authentication
	return fmt.Errorf("GCR authentication not implemented")
}

func (g *GCRClient) ScanImage(image string) (*ScanResult, error) {
	if g.mock {
		return g.mockClient.ScanImage(image)
	}
	// TODO: Implement real GCR image scanning
	return nil, fmt.Errorf("GCR image scanning not implemented")
}

func (g *GCRClient) ListRepositories() ([]string, error) {
	if g.mock {
		return g.mockClient.ListRepositories()
	}
	// TODO: Implement real GCR repository listing
	return nil, fmt.Errorf("GCR repository listing not implemented")
}

func (g *GCRClient) TestConnection() error {
	if g.mock {
		return g.mockClient.TestConnection()
	}
	// TODO: Implement real GCR connection test
	return fmt.Errorf("GCR connection test not implemented")
}

func (g *GCRClient) GetRegistryType() RegistryType {
	return GCRRegistry
}

// ECR Client
type ECRClient struct {
	accessKey  string
	secretKey  string
	mock       bool
	mockClient *mockRegistryClient
}

func NewECRClient() (*ECRClient, error) {
	if !testenv.HasAWSCredentials() || testenv.ShouldUseMockServices() {
		return &ECRClient{
			mock:       true,
			mockClient: newMockRegistryClient(ECRRegistry),
		}, nil
	}

	return &ECRClient{
		accessKey: testenv.AWSAccessKeyID.Setting(),
		secretKey: testenv.AWSSecretAccessKey.Setting(),
		mock:      false,
	}, nil
}

func (e *ECRClient) AuthenticateRegistry() error {
	if e.mock {
		return e.mockClient.AuthenticateRegistry()
	}
	return fmt.Errorf("ECR authentication not implemented")
}

func (e *ECRClient) ScanImage(image string) (*ScanResult, error) {
	if e.mock {
		return e.mockClient.ScanImage(image)
	}
	return nil, fmt.Errorf("ECR image scanning not implemented")
}

func (e *ECRClient) ListRepositories() ([]string, error) {
	if e.mock {
		return e.mockClient.ListRepositories()
	}
	return nil, fmt.Errorf("ECR repository listing not implemented")
}

func (e *ECRClient) TestConnection() error {
	if e.mock {
		return e.mockClient.TestConnection()
	}
	return fmt.Errorf("ECR connection test not implemented")
}

func (e *ECRClient) GetRegistryType() RegistryType {
	return ECRRegistry
}

// ACR Client
type ACRClient struct {
	clientID     string
	clientSecret string
	tenantID     string
	mock         bool
	mockClient   *mockRegistryClient
}

func NewACRClient() (*ACRClient, error) {
	if !testenv.HasAzureCredentials() || testenv.ShouldUseMockServices() {
		return &ACRClient{
			mock:       true,
			mockClient: newMockRegistryClient(ACRRegistry),
		}, nil
	}

	return &ACRClient{
		clientID:     testenv.AzureClientID.Setting(),
		clientSecret: testenv.AzureClientSecret.Setting(),
		tenantID:     testenv.AzureTenantID.Setting(),
		mock:         false,
	}, nil
}

func (a *ACRClient) AuthenticateRegistry() error {
	if a.mock {
		return a.mockClient.AuthenticateRegistry()
	}
	return fmt.Errorf("ACR authentication not implemented")
}

func (a *ACRClient) ScanImage(image string) (*ScanResult, error) {
	if a.mock {
		return a.mockClient.ScanImage(image)
	}
	return nil, fmt.Errorf("ACR image scanning not implemented")
}

func (a *ACRClient) ListRepositories() ([]string, error) {
	if a.mock {
		return a.mockClient.ListRepositories()
	}
	return nil, fmt.Errorf("ACR repository listing not implemented")
}

func (a *ACRClient) TestConnection() error {
	if a.mock {
		return a.mockClient.TestConnection()
	}
	return fmt.Errorf("ACR connection test not implemented")
}

func (a *ACRClient) GetRegistryType() RegistryType {
	return ACRRegistry
}

// Quay Client
type QuayClient struct {
	username   string
	password   string
	mock       bool
	mockClient *mockRegistryClient
}

func NewQuayClient() (*QuayClient, error) {
	if !testenv.HasRegistryCredentials() || testenv.ShouldUseMockServices() {
		return &QuayClient{
			mock:       true,
			mockClient: newMockRegistryClient(QuayRegistry),
		}, nil
	}

	return &QuayClient{
		username: testenv.RegistryUsername.Setting(),
		password: testenv.RegistryPassword.Setting(),
		mock:     false,
	}, nil
}

func (q *QuayClient) AuthenticateRegistry() error {
	if q.mock {
		return q.mockClient.AuthenticateRegistry()
	}
	return fmt.Errorf("Quay authentication not implemented")
}

func (q *QuayClient) ScanImage(image string) (*ScanResult, error) {
	if q.mock {
		return q.mockClient.ScanImage(image)
	}
	return nil, fmt.Errorf("Quay image scanning not implemented")
}

func (q *QuayClient) ListRepositories() ([]string, error) {
	if q.mock {
		return q.mockClient.ListRepositories()
	}
	return nil, fmt.Errorf("Quay repository listing not implemented")
}

func (q *QuayClient) TestConnection() error {
	if q.mock {
		return q.mockClient.TestConnection()
	}
	return fmt.Errorf("Quay connection test not implemented")
}

func (q *QuayClient) GetRegistryType() RegistryType {
	return QuayRegistry
}

// Red Hat Client
type RedHatClient struct {
	username   string
	password   string
	mock       bool
	mockClient *mockRegistryClient
}

func NewRedHatClient() (*RedHatClient, error) {
	if !testenv.HasRedHatCredentials() || testenv.ShouldUseMockServices() {
		return &RedHatClient{
			mock:       true,
			mockClient: newMockRegistryClient(RedHatRegistry),
		}, nil
	}

	return &RedHatClient{
		username: testenv.RedHatUsername.Setting(),
		password: testenv.RedHatPassword.Setting(),
		mock:     false,
	}, nil
}

func (r *RedHatClient) AuthenticateRegistry() error {
	if r.mock {
		return r.mockClient.AuthenticateRegistry()
	}
	return fmt.Errorf("Red Hat authentication not implemented")
}

func (r *RedHatClient) ScanImage(image string) (*ScanResult, error) {
	if r.mock {
		return r.mockClient.ScanImage(image)
	}
	return nil, fmt.Errorf("Red Hat image scanning not implemented")
}

func (r *RedHatClient) ListRepositories() ([]string, error) {
	if r.mock {
		return r.mockClient.ListRepositories()
	}
	return nil, fmt.Errorf("Red Hat repository listing not implemented")
}

func (r *RedHatClient) TestConnection() error {
	if r.mock {
		return r.mockClient.TestConnection()
	}
	return fmt.Errorf("Red Hat connection test not implemented")
}

func (r *RedHatClient) GetRegistryType() RegistryType {
	return RedHatRegistry
}

// Mock Registry Client Implementation (private)
type mockRegistryClient struct {
	registryType RegistryType
	responses    map[string]*ScanResult
}

func newMockRegistryClient(registryType RegistryType) *mockRegistryClient {
	return &mockRegistryClient{
		registryType: registryType,
		responses:    generateMockResponses(registryType),
	}
}

func (m *mockRegistryClient) AuthenticateRegistry() error {
	return nil
}

func (m *mockRegistryClient) ScanImage(image string) (*ScanResult, error) {
	if result, exists := m.responses[image]; exists {
		return result, nil
	}

	return &ScanResult{
		Image:           image,
		Vulnerabilities: generateMockVulnerabilities(image),
		ScanTime:        time.Now(),
		Status:          "SUCCESS",
		Components:      generateMockComponents(image),
	}, nil
}

func (m *mockRegistryClient) ListRepositories() ([]string, error) {
	switch m.registryType {
	case GCRRegistry:
		return []string{"gcr.io/test/app1", "gcr.io/test/app2"}, nil
	case ECRRegistry:
		return []string{"123456789.dkr.ecr.us-east-1.amazonaws.com/test1"}, nil
	case ACRRegistry:
		return []string{"testregistry.azurecr.io/app1"}, nil
	case QuayRegistry:
		return []string{"quay.io/test/app1", "quay.io/test/app2"}, nil
	case RedHatRegistry:
		return []string{"registry.redhat.io/rhel8/httpd-24"}, nil
	default:
		return []string{"mock.registry.com/test/app"}, nil
	}
}

func (m *mockRegistryClient) TestConnection() error {
	return nil
}

func (m *mockRegistryClient) GetRegistryType() RegistryType {
	return m.registryType
}

// Helper functions for mock data generation
func generateMockResponses(registryType RegistryType) map[string]*ScanResult {
	responses := make(map[string]*ScanResult)

	testImages := []string{
		"nginx:latest",
		"alpine:latest",
		"ubuntu:20.04",
		"vulnerable:latest",
		"secure:latest",
	}

	for _, image := range testImages {
		responses[image] = &ScanResult{
			Image:           image,
			Vulnerabilities: generateMockVulnerabilities(image),
			ScanTime:        time.Now(),
			Status:          "SUCCESS",
			Components:      generateMockComponents(image),
		}
	}

	return responses
}

func generateMockVulnerabilities(image string) []Vulnerability {
	if image == "secure:latest" {
		return []Vulnerability{}
	}

	vulnerabilities := []Vulnerability{
		{
			CVE:         "CVE-2023-1234",
			Severity:    "HIGH",
			CVSS:        7.5,
			Description: "Mock vulnerability for testing",
			Package:     "mock-package",
			Version:     "1.0.0",
			FixedBy:     "1.0.1",
		},
	}

	if image == "vulnerable:latest" {
		vulnerabilities = append(vulnerabilities, Vulnerability{
			CVE:         "CVE-2023-5678",
			Severity:    "CRITICAL",
			CVSS:        9.8,
			Description: "Critical mock vulnerability",
			Package:     "critical-package",
			Version:     "2.0.0",
			FixedBy:     "2.1.0",
		})
	}

	return vulnerabilities
}

func generateMockComponents(image string) []Component {
	return []Component{
		{
			Name:    "mock-component",
			Version: "1.0.0",
			Type:    "library",
		},
		{
			Name:    "base-os",
			Version: "20.04",
			Type:    "operating-system",
		},
	}
}
