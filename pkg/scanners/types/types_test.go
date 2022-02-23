package types

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

var _ Scanner = (*nopScanner)(nil)

type nopScanner struct{}

func (*nopScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted          { return nil }
func (*nopScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error)     { return nil, nil }
func (*nopScanner) Match(_ *storage.ImageName) bool                          { return false }
func (*nopScanner) Test() error                                              { return nil }
func (*nopScanner) Type() string                                             { return "" }
func (*nopScanner) Name() string                                             { return "" }
func (*nopScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) { return nil, nil }

var _ ImageScanner = (*nopImageScanner)(nil)
var _ ScannerGetter = (*nopImageScanner)(nil)

type nopImageScanner struct{ Scanner }

func (*nopImageScanner) DataSource() *storage.DataSource { return nil }
func (i *nopImageScanner) GetScanner() Scanner           { return i.Scanner }

var _ AsyncScanner = (*nopAsyncScanner)(nil)

type nopAsyncScanner struct{}

func (*nopAsyncScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted          { return nil }
func (*nopAsyncScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error)     { return nil, nil }
func (*nopAsyncScanner) Match(_ *storage.ImageName) bool                          { return false }
func (*nopAsyncScanner) Test() error                                              { return nil }
func (*nopAsyncScanner) Type() string                                             { return "" }
func (*nopAsyncScanner) Name() string                                             { return "" }
func (*nopAsyncScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) { return nil, nil }
func (*nopAsyncScanner) GetOrTriggerScan(_ *storage.Image) (*storage.ImageScan, error) {
	return nil, nil
}

func TestAsyncScannerAssertion(t *testing.T) {
	i := &nopImageScanner{Scanner: &nopAsyncScanner{}}

	shouldFail := func(i ImageScanner) {
		scanner, ok := i.(AsyncScanner)
		assert.Nil(t, scanner)
		assert.False(t, ok)
	}
	shouldFail(i)

	shouldPass := func(i ImageScanner) {
		scanner, ok := i.(ScannerGetter)
		assert.NotNil(t, scanner)
		assert.True(t, ok)

		var async AsyncScanner
		async, ok = scanner.GetScanner().(AsyncScanner)
		assert.NotNil(t, async)
		assert.True(t, ok)
	}
	shouldPass(i)
}
