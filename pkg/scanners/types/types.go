package types

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// Scanner type strings.
const (
	Clair     = "clair"
	Clairify  = "clairify"
	ClairV4   = "clairV4"
	Google    = "google"
	Quay      = "quay"
	ScannerV4 = "scannerv4"
)

// Scanner is the interface that all scanners must implement
type Scanner interface {
	ScanSemaphore

	// GetScan gets the scan for the given image.
	// It is a blocking call; if the scanner has not scanned the image yet,
	// the function blocks until it does. It returns an error if it fails to do so.
	GetScan(ctx context.Context, image *storage.Image) (*storage.ImageScan, error)
	Match(image *storage.ImageName) bool
	Test() error
	Type() string
	Name() string
	GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error)
}

// SBOM is the interface that contains the StackRox SBOM methods
type SBOMer interface {
	// GetSBOM to get sbom for an image
	GetSBOM(image *storage.Image) ([]byte, bool, error)
}

// ScannerSBOMer represents a Scanner with SBOM generation capabilities. This
// was initially created for mock generation to simplify tests.
//
//go:generate mockgen-wrapper
type ScannerSBOMer interface {
	Scanner
	SBOMer
}

// ImageScannerWithDataSource provides a GetScanner to retrieve the underlying Scanner and
// a DataSource function to describe which integration formed the interface.
//
//go:generate mockgen-wrapper
type ImageScannerWithDataSource interface {
	GetScanner() Scanner
	DataSource() *storage.DataSource
}

// ImageVulnerabilityGetter is a scanner which can retrieve vulnerabilities
// which exist in the given image components and the scan notes for the given image.
type ImageVulnerabilityGetter interface {
	GetVulnerabilities(image *storage.Image, components *ScanComponents, notes []scannerV1.Note) (*storage.ImageScan, error)
}

// NodeScanner is the interface all node scanners must implement
type NodeScanner interface {
	NodeScanSemaphore
	Name() string
	GetNodeInventoryScan(node *storage.Node, inv *storage.NodeInventory, ir *v4.IndexReport) (*storage.NodeScan, error)
	GetNodeScan(node *storage.Node) (*storage.NodeScan, error)
	TestNodeScanner() error
	Type() string
}

// NodeScannerWithDataSource provides a GetNodeScanner to retrieve the underlying NodeScanner and
// a DataSource function to describe which integration formed the interface.
type NodeScannerWithDataSource interface {
	GetNodeScanner() NodeScanner
	DataSource() *storage.DataSource
}

// OrchestratorScanner is the interface all orchestrator scanners must implement
type OrchestratorScanner interface {
	ScanSemaphore
	Name() string
	Type() string
	KubernetesScan(string) (map[string][]*storage.EmbeddedVulnerability, error)
	OpenShiftScan(string) ([]*storage.EmbeddedVulnerability, error)
	IstioScan(string) ([]*storage.EmbeddedVulnerability, error)
}
