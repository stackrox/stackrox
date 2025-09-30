package external

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/testutils/credentials"
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

// NewRegistryClient creates a registry client based on available credentials
func NewRegistryClient(creds *credentials.Credentials, registryType RegistryType) (RegistryClient, error) {
	// Check if we should use mocks
	if creds.ShouldUseMockServices() {
		return NewMockRegistryClient(registryType), nil
	}

	// Create real client based on type and available credentials
	switch registryType {
	case GCRRegistry:
		if !creds.HasGCRCredentials() {
			if creds.IsDevelopmentMode() {
				return NewMockRegistryClient(GCRRegistry), nil
			}
			return nil, fmt.Errorf("GCR credentials required")
		}
		return NewGCRClient(creds.GoogleGCRCredentials)

	case ECRRegistry:
		if !creds.HasAWSCredentials() {
			if creds.IsDevelopmentMode() {
				return NewMockRegistryClient(ECRRegistry), nil
			}
			return nil, fmt.Errorf("AWS credentials required")
		}
		return NewECRClient(creds.AWSAccessKeyID, creds.AWSSecretAccessKey)

	case ACRRegistry:
		if !creds.HasAzureCredentials() {
			if creds.IsDevelopmentMode() {
				return NewMockRegistryClient(ACRRegistry), nil
			}
			return nil, fmt.Errorf("Azure credentials required")
		}
		return NewACRClient(creds.AzureClientID, creds.AzureClientSecret, creds.AzureTenantID)

	case QuayRegistry:
		if !creds.HasRegistryCredentials() {
			if creds.IsDevelopmentMode() {
				return NewMockRegistryClient(QuayRegistry), nil
			}
			return nil, fmt.Errorf("Quay credentials required")
		}
		return NewQuayClient(creds.RegistryUsername, creds.RegistryPassword)

	case RedHatRegistry:
		if !creds.HasRedHatCredentials() {
			if creds.IsDevelopmentMode() {
				return NewMockRegistryClient(RedHatRegistry), nil
			}
			return nil, fmt.Errorf("Red Hat credentials required")
		}
		return NewRedHatClient(creds.RedHatUsername, creds.RedHatPassword)

	default:
		return nil, fmt.Errorf("unsupported registry type: %s", registryType)
	}
}

// GetAvailableRegistries returns a list of registry clients that can be created with current credentials
func GetAvailableRegistries(creds *credentials.Credentials) []RegistryClient {
	var clients []RegistryClient

	registryTypes := []RegistryType{
		GCRRegistry, ECRRegistry, ACRRegistry, QuayRegistry, RedHatRegistry,
	}

	for _, regType := range registryTypes {
		client, err := NewRegistryClient(creds, regType)
		if err == nil {
			clients = append(clients, client)
		}
	}

	return clients
}

// Mock Registry Client Implementation
type MockRegistryClient struct {
	registryType RegistryType
	responses    map[string]*ScanResult
}

func NewMockRegistryClient(registryType RegistryType) *MockRegistryClient {
	return &MockRegistryClient{
		registryType: registryType,
		responses:    generateMockResponses(registryType),
	}
}

func (m *MockRegistryClient) AuthenticateRegistry() error {
	// Mock authentication always succeeds
	return nil
}

func (m *MockRegistryClient) ScanImage(image string) (*ScanResult, error) {
	// Return predefined mock data based on image name
	if result, exists := m.responses[image]; exists {
		return result, nil
	}

	// Generate mock vulnerabilities based on image name patterns
	return &ScanResult{
		Image:           image,
		Vulnerabilities: generateMockVulnerabilities(image),
		ScanTime:        time.Now(),
		Status:          "SUCCESS",
		Components:      generateMockComponents(image),
	}, nil
}

func (m *MockRegistryClient) ListRepositories() ([]string, error) {
	// Return mock repository list based on registry type
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

func (m *MockRegistryClient) TestConnection() error {
	// Mock connection test always succeeds
	return nil
}

func (m *MockRegistryClient) GetRegistryType() RegistryType {
	return m.registryType
}

// Helper functions for mock data generation
func generateMockResponses(registryType RegistryType) map[string]*ScanResult {
	responses := make(map[string]*ScanResult)

	// Add common test images with predefined vulnerabilities
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
	// Generate different vulnerability patterns based on image name
	if image == "secure:latest" {
		return []Vulnerability{} // No vulnerabilities for secure image
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
		// Add more vulnerabilities for vulnerable test image
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

// Real registry client stubs - these would be implemented with actual SDK calls

// GCR Client
type GCRClient struct {
	credentials string
}

func NewGCRClient(credentials string) (*GCRClient, error) {
	return &GCRClient{credentials: credentials}, nil
}

func (g *GCRClient) AuthenticateRegistry() error {
	// TODO: Implement real GCR authentication
	return fmt.Errorf("GCR authentication not implemented")
}

func (g *GCRClient) ScanImage(image string) (*ScanResult, error) {
	// TODO: Implement real GCR image scanning
	return nil, fmt.Errorf("GCR image scanning not implemented")
}

func (g *GCRClient) ListRepositories() ([]string, error) {
	// TODO: Implement real GCR repository listing
	return nil, fmt.Errorf("GCR repository listing not implemented")
}

func (g *GCRClient) TestConnection() error {
	// TODO: Implement real GCR connection test
	return fmt.Errorf("GCR connection test not implemented")
}

func (g *GCRClient) GetRegistryType() RegistryType {
	return GCRRegistry
}

// ECR Client stub
type ECRClient struct {
	accessKey string
	secretKey string
}

func NewECRClient(accessKey, secretKey string) (*ECRClient, error) {
	return &ECRClient{accessKey: accessKey, secretKey: secretKey}, nil
}

func (e *ECRClient) AuthenticateRegistry() error {
	return fmt.Errorf("ECR authentication not implemented")
}

func (e *ECRClient) ScanImage(image string) (*ScanResult, error) {
	return nil, fmt.Errorf("ECR image scanning not implemented")
}

func (e *ECRClient) ListRepositories() ([]string, error) {
	return nil, fmt.Errorf("ECR repository listing not implemented")
}

func (e *ECRClient) TestConnection() error {
	return fmt.Errorf("ECR connection test not implemented")
}

func (e *ECRClient) GetRegistryType() RegistryType {
	return ECRRegistry
}

// ACR Client stub
type ACRClient struct {
	clientID     string
	clientSecret string
	tenantID     string
}

func NewACRClient(clientID, clientSecret, tenantID string) (*ACRClient, error) {
	return &ACRClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		tenantID:     tenantID,
	}, nil
}

func (a *ACRClient) AuthenticateRegistry() error {
	return fmt.Errorf("ACR authentication not implemented")
}

func (a *ACRClient) ScanImage(image string) (*ScanResult, error) {
	return nil, fmt.Errorf("ACR image scanning not implemented")
}

func (a *ACRClient) ListRepositories() ([]string, error) {
	return nil, fmt.Errorf("ACR repository listing not implemented")
}

func (a *ACRClient) TestConnection() error {
	return fmt.Errorf("ACR connection test not implemented")
}

func (a *ACRClient) GetRegistryType() RegistryType {
	return ACRRegistry
}

// Quay Client stub
type QuayClient struct {
	username string
	password string
}

func NewQuayClient(username, password string) (*QuayClient, error) {
	return &QuayClient{username: username, password: password}, nil
}

func (q *QuayClient) AuthenticateRegistry() error {
	return fmt.Errorf("Quay authentication not implemented")
}

func (q *QuayClient) ScanImage(image string) (*ScanResult, error) {
	return nil, fmt.Errorf("Quay image scanning not implemented")
}

func (q *QuayClient) ListRepositories() ([]string, error) {
	return nil, fmt.Errorf("Quay repository listing not implemented")
}

func (q *QuayClient) TestConnection() error {
	return fmt.Errorf("Quay connection test not implemented")
}

func (q *QuayClient) GetRegistryType() RegistryType {
	return QuayRegistry
}

// Red Hat Client stub
type RedHatClient struct {
	username string
	password string
}

func NewRedHatClient(username, password string) (*RedHatClient, error) {
	return &RedHatClient{username: username, password: password}, nil
}

func (r *RedHatClient) AuthenticateRegistry() error {
	return fmt.Errorf("Red Hat authentication not implemented")
}

func (r *RedHatClient) ScanImage(image string) (*ScanResult, error) {
	return nil, fmt.Errorf("Red Hat image scanning not implemented")
}

func (r *RedHatClient) ListRepositories() ([]string, error) {
	return nil, fmt.Errorf("Red Hat repository listing not implemented")
}

func (r *RedHatClient) TestConnection() error {
	return fmt.Errorf("Red Hat connection test not implemented")
}

func (r *RedHatClient) GetRegistryType() RegistryType {
	return RedHatRegistry
}