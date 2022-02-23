package types

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
