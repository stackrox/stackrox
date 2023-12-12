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
	GetVulnerabilities(image *storage.Image, components *scannerV1.Components, notes []scannerV1.Note) (*storage.ImageScan, error)
}

// NodeScanner is the interface all node scanners must implement
type NodeScanner interface {
	NodeScanSemaphore
	Name() string
	GetNodeInventoryScan(node *storage.Node, inv *storage.NodeInventory) (*storage.NodeScan, error)
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
