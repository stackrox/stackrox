package types

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// Scanner is the interface that all scanners must implement
type Scanner interface {
	ScanSemaphore

	// GetScan gets the scan for the given image.
	// It is a blocking call; if the scanner has not scanned the image yet,
	// the function blocks until it does. It returns an error if it fails to do so.
	GetScan(image *storage.Image) (*storage.ImageScan, error)
	Match(image *storage.ImageName) bool
	Test() error
	Type() string
	Name() string
	GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error)
}

// ImageScanner adds a DataSource function to Scanner that describes which
// integration formed the interface
type ImageScanner interface {
	Scanner
	DataSource() *storage.DataSource
}

// ImageVulnerabilityGetter is a scanner which can retrieve vulnerabilities
// which exist in the given image components and the scan notes for the given image.
type ImageVulnerabilityGetter interface {
	GetVulnerabilities(image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (*storage.ImageScan, error)
}

// AsyncScanner is an image scanner that can be accessed asynchronously.
type AsyncScanner interface {
	Scanner
	// GetOrTriggerScan does a non-blocking request to the scanner.
	// It gets the scan for the given image if it exists;
	// if not, implementations trigger a new one and instantly return.
	GetOrTriggerScan(image *storage.Image) (*storage.ImageScan, error)
}

// NodeScanner is the interface all node scanners must implement
type NodeScanner interface {
	NodeScanSemaphore
	Name() string
	GetNodeScan(node *storage.Node) (*storage.NodeScan, error)
	TestNodeScanner() error
	Type() string
}

// NodeScannerWithDataSource adds a DataSource function to NodeScanner that describes which
// integration formed the interface
type NodeScannerWithDataSource interface {
	NodeScanner
	DataSource() *storage.DataSource
}

// OrchestratorScanner is the interface all orchestrator scanners must implement
type OrchestratorScanner interface {
	ScanSemaphore
	Name() string
	Type() string
	KubernetesScan(string) (map[string][]*storage.EmbeddedVulnerability, error)
	OpenShiftScan(string) ([]*storage.EmbeddedVulnerability, error)
}
