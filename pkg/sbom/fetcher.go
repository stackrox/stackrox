package sbom

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	gcrRemote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/pkg/oci"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protoconv"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sliceutils"
	"golang.org/x/time/rate"
)

/*
Fetcher
	- fetching should, for now, be limited the "cosign" way, but potentially we could do this differently as well.
	- _could_ include a client abstraction to handle different sbom types (i.e. spdx, cyclonedx, syft) to obtain contents (iff they are helpful).
	- do we require the SBOM media type (i.e. in which format the SBOM is) at all or not? Would a policy make sense
      to alert on SBOMs that do not match a specific media type / format (e.g. I only want syft SBOMs formats)?

Note: should make this a "sigstore" client instead, which implements different interfaces, i.e.
	-> fetching signatures & sboms
	-> verifying signatures & sboms
	-> potentially anything else.

	-> the client itself could be created by a builder to specify anything that is specifically required, i.e.:
		-> cosign keyless vs cosign pubkey
		-> different formats of SBOMs
*/

var (
	insecureDefaultTransport *http.Transport

	_ Fetcher = (*sigstoreSBOMFetcher)(nil)
)

const (
	imageSBOMScopeAnnotationKey = "dev.sigstore.sbom.scope"

	scopeAll   = "all"
	scopeLayer = "layer="
	scopePath  = "path="
)

func init() {
	insecureDefaultTransport = gcrRemote.DefaultTransport.Clone()
	insecureDefaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

type sigstoreSBOMFetcher struct {
	registryRateLimiter *rate.Limiter
}

func newSigstoreSBOMFetcher() *sigstoreSBOMFetcher {
	return &sigstoreSBOMFetcher{
		registryRateLimiter: rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
	}
}

func (s *sigstoreSBOMFetcher) FetchSBOM(ctx context.Context, image *storage.Image,
	registry registryTypes.Registry) (*storage.ImageSBOM, error) {
	imgFullName := image.GetName().GetFullName()

	imgRef, err := name.ParseReference(imgFullName)
	if err != nil {
		return nil, err
	}

	if err = s.registryRateLimiter.Wait(ctx); err != nil {
		return nil, errors.Wrap(err, "waiting for rate limiter entrance")
	}

	// Fetch the signed entity - in our case the image.
	se, err := ociremote.SignedEntity(imgRef, ociremote.WithRemoteOptions(optionsFromRegistry(registry)...))
	if err != nil {
		return nil, errors.Wrapf(err, "fetching image %q", imgFullName)
	}

	_, isIndex := se.(oci.SignedImageIndex)

	sbom, err := se.Attachment("sbom")

	// Probs not the one we are looking for - need to use our own error mapping (heroku + ggcr).
	if checkIfErrorContainsCode(err, http.StatusNotFound) {
		// No sbom attached to the image reference.
		if !isIndex {
			return nil, nil
		}
	} else if checkIfErrorContainsCode(err, http.StatusUnauthorized) {
		return nil, errox.NotAuthorized.CausedBy(err)
	} else if err != nil {
		return nil, err
	}

	sboms, err := getSBOMsFromScope(sbom)
	if err != nil {
		return nil, err
	}

	return &storage.ImageSBOM{
		Fetched: protoconv.ConvertTimeToTimestamp(time.Now()),
		Sboms:   sboms,
	}, nil
}

func getSBOMsFromScope(sbom oci.File) ([]*storage.SBOM, error) {
	manifest, err := sbom.Manifest()
	if err != nil {
		return nil, err
	}

	// Check if the "scope" annotation is set.
	// The scope annotation is used to determine for which specific part of the image the SBOM covers.
	// The following scopes can be expected:
	// 	- if no scope is set, we assume the SBOM covers the whole contents of the image.
	//	- if scope is set and contains "all", the SBOM covers the whole contents of the image.
	// 	- if scope is set and specifies a specific layer, i.e. layer=sha256:$DIGEST, the SBOM covers a specific layer of the image.
	//	- if scope is set and specifies a specific paht, i.e. path=<some/path>, the SBOM covers a specific file at the path in the flattened container image.
	// For more, see here: https://github.com/sigstore/cosign/blob/main/specs/SBOM_SPEC.md#scopes
	scope, exists := manifest.Annotations[imageSBOMScopeAnnotationKey]
	if !exists {
		return []*storage.SBOM{
			{
				SBOM: &storage.SBOM_CompleteSbom{},
				Type: storage.SBOM_COMPLETE_SBOM,
			},
		}, nil
	}
	// Need to split the string, since the scope _may_ be repeated, where "," will be used as separator.
	scopeValues := strings.Split(scope, ",")

	var layers, paths []string
	// Traverse through the scope values. Since scope values _may_ be repeated, we will add a list of all referenced
	// layers / paths.
	for _, scopeVal := range scopeValues {
		switch {
		case strings.HasPrefix(scopeVal, scopeLayer):
			layers = append(layers, strings.TrimPrefix(scopeVal, scopeLayer))
		case strings.HasPrefix(scopeVal, scopePath):
			paths = append(paths, strings.TrimPrefix(scopeVal, scopePath))
		case scopeVal == scopeAll:
			return []*storage.SBOM{
				{
					SBOM: &storage.SBOM_CompleteSbom{},
					Type: storage.SBOM_COMPLETE_SBOM,
				},
			}, nil
		}
	}

	var sboms []*storage.SBOM

	if len(layers) == 0 && len(paths) == 0 {
		return nil, errox.InvariantViolation.New("expected at least one SBOM scoped to either a file or layer")
	}

	if len(layers) > 0 {
		sboms = append(sboms, &storage.SBOM{
			SBOM: &storage.SBOM_LayerSbom{LayerSbom: &storage.LayerSBOM{
				ReferencedImageLayerSha: layers,
			}},
			Type: storage.SBOM_LAYER_SCOPED_SBOM,
		})
	}

	if len(paths) > 0 {
		sboms = append(sboms, &storage.SBOM{
			SBOM: &storage.SBOM_FileSbom{FileSbom: &storage.FileSBOM{
				PathInImage: paths,
			}},
			Type: storage.SBOM_FILE_SCOPED_SBOM,
		})
	}

	return sboms, nil
}

// COPIED FROM PKG/SIGNATURES, SHOULD BE SHARED IN GENERIC CLIENT.
func optionsFromRegistry(registry registryTypes.Registry) []gcrRemote.Option {
	registryCfg := registry.Config()
	if registryCfg == nil {
		return nil
	}

	var opts []gcrRemote.Option
	// Only create an option for the transport if we have custom authentication. By default, the transport will assume
	// anonymous access. We need to check both values since some registries use special strings for username, i.e.
	// GCR will use "_json_key" if used with a service account, quay.io registries will use "$oauthtoken".
	if registryCfg.Username != "" && registryCfg.Password != "" {
		// We are not changing the transport but keep default values, so it is safe to assign it to a variable without
		// potential race conditions.
		tr := gcrRemote.DefaultTransport
		if registryCfg.Insecure {
			tr = insecureDefaultTransport
		}
		// Instead of relying on gcrRemote's authentication, we are using the same authentication we already use for
		// our registries. The wrapped transport will ensure we are authenticated properly with all currently supported
		// registries. Ideally, we would in general use the same libraries for both, but cosign doesn't support
		// exchanging gcrRemote for now (we could however move to gcrRemote within the registry as well).
		opts = append(opts, gcrRemote.WithTransport(
			dockerRegistry.WrapTransport(tr, strings.TrimSuffix(registryCfg.URL, "/"),
				registryCfg.Username, registryCfg.Password)))
	}
	return opts
}

// checkIfErrorContainsCode will try retrieve a http.StatusCode from the given error by casting the error to either
// transport.Error or registry.HttpStatusError and check whether the code is contained within a given list of codes.
// In case the error is matching one of these types and the code is contained within the given codes, true will be
// returned.
// If the error is not matching any of these types or the code is not contained in the given codes, false will be
// returned.
func checkIfErrorContainsCode(err error, codes ...int) bool {
	var transportErr *transport.Error
	var statusError *dockerRegistry.HttpStatusError

	// Transport error is returned by go-containerregistry for any errors occurred post authentication.
	if errors.As(err, &transportErr) {
		return sliceutils.IntFind(codes, transportErr.StatusCode) != -1
	}

	// HttpStatusError is returned by heroku-client for any errors occurred during authentication.
	if errors.As(err, &statusError) && statusError.Response != nil {
		return sliceutils.IntFind(codes, statusError.Response.StatusCode) != -1
	}

	return false
}
