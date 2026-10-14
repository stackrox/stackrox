package updater

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/rhel/vex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configCaptureFactory captures the callback registered with the updates manager.
type configCaptureFactory struct {
	driver.UpdaterSetFactory
	configure driver.ConfigUnmarshaler
}

func (f *configCaptureFactory) Configure(_ context.Context, configure driver.ConfigUnmarshaler, _ *http.Client) error {
	f.configure = configure
	return nil
}

func rhelVexConfigForTest(t *testing.T) driver.ConfigUnmarshaler {
	t.Helper()
	factory := &configCaptureFactory{}
	opts := append(rhelVexOpts(), updates.WithFactories(map[string]driver.UpdaterSetFactory{
		rhelVexUpdaterName: factory,
	}))
	_, err := updates.NewManager(context.Background(), nil, nil, http.DefaultClient, opts...)
	require.NoError(t, err)
	require.NotNil(t, factory.configure)
	return factory.configure
}

func TestRHELVexOpts(t *testing.T) {
	configure := rhelVexConfigForTest(t)
	for name, tc := range map[string]struct {
		value string
		want  bool
		err   bool
	}{
		"unset":    {},
		"enabled":  {value: "true", want: true},
		"disabled": {value: "false"},
		"invalid":  {value: "invalid", err: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("STACKROX_RHEL_VEX_IGNORE_KERNEL_PACKAGES", tc.value)
			t.Setenv("STACKROX_RHEL_VEX_COMPRESSED_FILE_TIMEOUT", "10m")
			factory := &vex.FactoryConfig{}
			updater := &vex.UpdaterConfig{}
			for _, cfg := range []any{factory, updater} {
				err := configure(cfg)
				if tc.err {
					require.ErrorContains(t, err, "STACKROX_RHEL_VEX_IGNORE_KERNEL_PACKAGES")
				} else {
					require.NoError(t, err)
				}
			}
			if !tc.err {
				assert.False(t, factory.IgnoreKernelPackages)
				assert.Equal(t, tc.want, updater.IgnoreKernelPackages)
				assert.Equal(t, claircore.Duration(10*time.Minute), factory.CompressedFileTimeout)
			}
		})
	}
}

func TestRHELVexOptsTimeout(t *testing.T) {
	configure := rhelVexConfigForTest(t)
	for name, tc := range map[string]struct {
		value string
		want  time.Duration
	}{
		"empty preserves default":   {want: time.Minute},
		"invalid preserves default": {value: "invalid", want: time.Minute},
		"valid overrides default":   {value: "10m", want: 10 * time.Minute},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("STACKROX_RHEL_VEX_COMPRESSED_FILE_TIMEOUT", tc.value)
			t.Setenv("STACKROX_RHEL_VEX_IGNORE_KERNEL_PACKAGES", "true")
			factory := &vex.FactoryConfig{CompressedFileTimeout: claircore.Duration(time.Minute)}
			require.NoError(t, configure(factory))
			assert.Equal(t, claircore.Duration(tc.want), factory.CompressedFileTimeout)
			updater := &vex.UpdaterConfig{}
			require.NoError(t, configure(updater))
			assert.True(t, updater.IgnoreKernelPackages)
		})
	}
}

func TestFilterSources(t *testing.T) {
	bundles := map[string][]updates.ManagerOption{
		"alpha":   nil,
		"bravo":   nil,
		"charlie": nil,
	}

	tests := map[string]struct {
		selected  []string
		wantKeys  []string
		wantError string
	}{
		"single source": {
			selected: []string{"alpha"},
			wantKeys: []string{"alpha"},
		},
		"multiple sources": {
			selected: []string{"alpha", "charlie"},
			wantKeys: []string{"alpha", "charlie"},
		},
		"unknown source": {
			selected:  []string{"alpha", "bogus"},
			wantError: `unknown source: "bogus"`,
		},
		"all sources": {
			selected: []string{"alpha", "bravo", "charlie"},
			wantKeys: []string{"alpha", "bravo", "charlie"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := filterSources(bundles, tc.selected)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantKeys, slices.Collect(maps.Keys(result)))
		})
	}
}
