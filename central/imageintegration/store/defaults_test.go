package store

import (
	"errors"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scanners"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/semaphore"
)

func TestGetDelayedIntegrations(t *testing.T) {
	tests := map[string]struct {
		legacyScannerEnabled bool
		expectCount          int
	}{
		"when LegacyScanner is enabled, returns Clairify scanner": {
			legacyScannerEnabled: true,
			expectCount:          1,
		},
		"when LegacyScanner is disabled, returns empty list": {
			legacyScannerEnabled: false,
			expectCount:          0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			testutils.MustUpdateFeature(t, features.LegacyScanner, tc.legacyScannerEnabled)

			integrations := GetDelayedIntegrations()

			if tc.expectCount == 0 {
				assert.Empty(t, integrations)
			} else {
				assert.Len(t, integrations, tc.expectCount)
				assert.Equal(t, defaultScanner.GetName(), integrations[0].Integration.GetName())
			}
		})
	}
}

type mockScanner struct {
	testErr error
}

func (m *mockScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted { return nil }
func (m *mockScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
	return nil, nil
}
func (m *mockScanner) Match(_ *storage.ImageName) bool { return false }
func (m *mockScanner) Test() error                     { return m.testErr }
func (m *mockScanner) Type() string                    { return "mock" }
func (m *mockScanner) Name() string                    { return "mock" }
func (m *mockScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}

type mockClosableScanner struct {
	mockScanner
	closeCalled bool
}

func (m *mockClosableScanner) Close() error {
	m.closeCalled = true
	return nil
}

func TestMakeDelayedIntegration(t *testing.T) {
	integration := &storage.ImageIntegration{Id: "test-id", Name: "test"}

	tests := map[string]struct {
		creatorErr    error
		testErr       error
		closable      bool
		expectTrigger bool
		expectClosed  bool
	}{
		"scanner test succeeds and Close is called": {
			closable:      true,
			expectTrigger: true,
			expectClosed:  true,
		},
		"scanner test fails and Close is called": {
			testErr:       errors.New("connection refused"),
			closable:      true,
			expectTrigger: false,
			expectClosed:  true,
		},
		"creator returns error so no scanner to close": {
			creatorErr:    errors.New("bad config"),
			expectTrigger: false,
			expectClosed:  false,
		},
		"scanner without io.Closer works without panic": {
			closable:      false,
			expectTrigger: true,
			expectClosed:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var closable *mockClosableScanner

			creatorFactory := func() scanners.Creator {
				return func(_ *storage.ImageIntegration) (scannerTypes.Scanner, error) {
					if tc.creatorErr != nil {
						return nil, tc.creatorErr
					}
					if tc.closable {
						closable = &mockClosableScanner{mockScanner: mockScanner{testErr: tc.testErr}}
						return closable, nil
					}
					return &mockScanner{testErr: tc.testErr}, nil
				}
			}

			di := makeDelayedIntegration(integration, creatorFactory)
			result := di.Trigger()

			assert.Equal(t, tc.expectTrigger, result)
			if tc.expectClosed {
				assert.True(t, closable.closeCalled, "Close() should have been called")
			}
		})
	}
}
