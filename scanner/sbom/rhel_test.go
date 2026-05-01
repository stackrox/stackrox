package sbom

import (
	"context"
	"testing"

	"github.com/package-url/packageurl-go"
	"github.com/quay/claircore/rhel"
	"github.com/stackrox/rox/pkg/scannerv4/repositorytocpe"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stackrox/rox/scanner/matcher/repo2cpe"
	"github.com/stackrox/rox/scanner/matcher/repo2cpe/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newUpdaterWithMapping(t *testing.T, mapping map[string]repositorytocpe.Repo) *repo2cpe.Updater {
	t.Helper()
	ctrl := gomock.NewController(t)
	g := mocks.NewMockGetter(ctrl)
	g.EXPECT().
		GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
		Return(&indexer.FetchResult{
			Modified: true,
			Data:     &repositorytocpe.MappingFile{Data: mapping},
		}, nil).
		AnyTimes()
	u := repo2cpe.NewUpdater(g)
	t.Cleanup(u.Close)
	// Trigger lazy init.
	_, err := u.Get(context.Background())
	require.NoError(t, err)
	return u
}

func TestNewRHELCPETransformFunc(t *testing.T) {
	ctx := context.Background()

	t.Run("enriches PURL with CPEs from mapping", func(t *testing.T) {
		u := newUpdaterWithMapping(t, map[string]repositorytocpe.Repo{
			"rhel-8-for-x86_64-baseos-rpms": {CPEs: []string{"cpe:/o:redhat:rhel:8", "cpe:/o:redhat:enterprise_linux:8"}},
		})
		tf := NewRHELCPETransformFunc(u)

		p := packageurl.PackageURL{
			Type:      rhel.PURLType,
			Namespace: rhel.PURLNamespace,
			Name:      "bash",
			Version:   "5.1.8-6.el8",
			Qualifiers: packageurl.QualifiersFromMap(map[string]string{
				rhel.PURLRepositoryID: "rhel-8-for-x86_64-baseos-rpms",
				"arch":                "x86_64",
			}),
		}

		err := tf(ctx, &p)
		require.NoError(t, err)

		qs := p.Qualifiers.Map()
		assert.Equal(t, "cpe:/o:redhat:rhel:8,cpe:/o:redhat:enterprise_linux:8", qs[rhel.PURLRepositoryCPEs])
	})

	t.Run("does not overwrite existing repository_cpes", func(t *testing.T) {
		u := newUpdaterWithMapping(t, map[string]repositorytocpe.Repo{
			"rhel-8-for-x86_64-baseos-rpms": {CPEs: []string{"cpe:/o:redhat:rhel:8"}},
		})
		tf := NewRHELCPETransformFunc(u)

		p := packageurl.PackageURL{
			Type:      rhel.PURLType,
			Namespace: rhel.PURLNamespace,
			Name:      "bash",
			Version:   "5.1.8-6.el8",
			Qualifiers: packageurl.QualifiersFromMap(map[string]string{
				rhel.PURLRepositoryID:   "rhel-8-for-x86_64-baseos-rpms",
				rhel.PURLRepositoryCPEs: "cpe:/o:redhat:original:8",
			}),
		}

		err := tf(ctx, &p)
		require.NoError(t, err)

		qs := p.Qualifiers.Map()
		assert.Equal(t, "cpe:/o:redhat:original:8", qs[rhel.PURLRepositoryCPEs])
	})

	t.Run("skips when no repository_id", func(t *testing.T) {
		u := newUpdaterWithMapping(t, map[string]repositorytocpe.Repo{
			"rhel-8-for-x86_64-baseos-rpms": {CPEs: []string{"cpe:/o:redhat:rhel:8"}},
		})
		tf := NewRHELCPETransformFunc(u)

		p := packageurl.PackageURL{
			Type:      rhel.PURLType,
			Namespace: rhel.PURLNamespace,
			Name:      "bash",
			Version:   "5.1.8-6.el8",
			Qualifiers: packageurl.QualifiersFromMap(map[string]string{
				"arch": "x86_64",
			}),
		}

		err := tf(ctx, &p)
		require.NoError(t, err)

		qs := p.Qualifiers.Map()
		_, has := qs[rhel.PURLRepositoryCPEs]
		assert.False(t, has)
	})

	t.Run("skips when repository_id not in mapping", func(t *testing.T) {
		u := newUpdaterWithMapping(t, map[string]repositorytocpe.Repo{
			"rhel-8-for-x86_64-baseos-rpms": {CPEs: []string{"cpe:/o:redhat:rhel:8"}},
		})
		tf := NewRHELCPETransformFunc(u)

		p := packageurl.PackageURL{
			Type:      rhel.PURLType,
			Namespace: rhel.PURLNamespace,
			Name:      "bash",
			Version:   "5.1.8-6.el8",
			Qualifiers: packageurl.QualifiersFromMap(map[string]string{
				rhel.PURLRepositoryID: "unknown-repo",
			}),
		}

		err := tf(ctx, &p)
		require.NoError(t, err)

		qs := p.Qualifiers.Map()
		_, has := qs[rhel.PURLRepositoryCPEs]
		assert.False(t, has)
	})

	t.Run("degrades gracefully when mapping unavailable", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		g := mocks.NewMockGetter(ctrl)
		g.EXPECT().
			GetRepositoryToCPEMapping(gomock.Any(), gomock.Any()).
			Return(nil, assert.AnError).
			AnyTimes()
		u := repo2cpe.NewUpdater(g)
		t.Cleanup(u.Close)
		// Trigger lazy init (will fail).
		_, _ = u.Get(context.Background())

		tf := NewRHELCPETransformFunc(u)

		p := packageurl.PackageURL{
			Type:      rhel.PURLType,
			Namespace: rhel.PURLNamespace,
			Name:      "bash",
			Version:   "5.1.8-6.el8",
			Qualifiers: packageurl.QualifiersFromMap(map[string]string{
				rhel.PURLRepositoryID: "rhel-8-for-x86_64-baseos-rpms",
			}),
		}

		err := tf(ctx, &p)
		require.NoError(t, err)

		qs := p.Qualifiers.Map()
		_, has := qs[rhel.PURLRepositoryCPEs]
		assert.False(t, has)
	})
}
