package indexer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/stackrox/rox/clair-adapter/clairclient"
)

// fetchManifestLayers fetches an image manifest from a container registry
// and returns layer descriptors suitable for Clair's indexing API.
func fetchManifestLayers(ctx context.Context, imageURL string, opts indexOpts) ([]clairclient.Layer, error) {
	// 1. Parse image reference
	ref, err := parseImageRef(imageURL)
	if err != nil {
		return nil, err
	}

	// 2. Build auth option
	var auth authn.Authenticator = authn.Anonymous
	if opts.username != "" {
		auth = &authn.Basic{Username: opts.username, Password: opts.password}
	}

	// 3. Fetch image descriptor to get layer list
	desc, err := remote.Get(ref, remote.WithAuth(auth), remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("fetching image descriptor: %w", err)
	}

	img, err := desc.Image()
	if err != nil {
		return nil, fmt.Errorf("getting image from descriptor: %w", err)
	}

	imgLayers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("getting image layers: %w", err)
	}

	// 4. Build authenticated HTTP client for layer URI construction
	httpClient, err := buildLayerHTTPClient(ctx, ref, auth, opts.insecureSkipTLSVerify)
	if err != nil {
		return nil, fmt.Errorf("building layer HTTP client: %w", err)
	}

	// 5. Construct layer descriptors for Clair
	var layers []clairclient.Layer
	for _, layer := range imgLayers {
		digest, err := layer.Digest()
		if err != nil {
			return nil, fmt.Errorf("getting layer digest: %w", err)
		}

		layerURI, headers, err := buildLayerURI(ctx, httpClient, ref, digest.String())
		if err != nil {
			return nil, fmt.Errorf("building layer URI for %s: %w", digest, err)
		}

		layers = append(layers, clairclient.Layer{
			Hash:    digest.String(),
			URI:     layerURI,
			Headers: headers,
		})
	}

	return layers, nil
}

// parseImageRef parses a container image URL into a name.Reference.
// Accepts formats: "docker.io/library/bash:5.1", "https://registry.example.com/repo:tag"
func parseImageRef(imageURL string) (name.Reference, error) {
	// Strip scheme if present
	u := imageURL
	if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		parsed, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("parsing URL: %w", err)
		}
		u = parsed.Host + parsed.Path
	}

	ref, err := name.ParseReference(u)
	if err != nil {
		return nil, fmt.Errorf("parsing image reference %q: %w", u, err)
	}
	return ref, nil
}

// buildLayerHTTPClient creates an HTTP client with registry authentication.
func buildLayerHTTPClient(ctx context.Context, ref name.Reference, auth authn.Authenticator, insecure bool) (*http.Client, error) {
	base, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("unexpected default transport type %T", http.DefaultTransport)
	}
	tr := base.Clone()
	if insecure {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
	}

	// Add retry wrapper
	var roundTripper http.RoundTripper = transport.NewRetry(tr)

	// Add authentication transport (handles registry challenge-response)
	repo := ref.Context()
	roundTripper, err := transport.NewWithContext(ctx, repo.Registry, auth, roundTripper, []string{repo.Scope(transport.PullScope)})
	if err != nil {
		return nil, fmt.Errorf("creating auth transport: %w", err)
	}

	return &http.Client{Transport: roundTripper}, nil
}

// buildLayerURI constructs the blob URI for a layer and captures auth headers.
// It sends a partial-content request to the registry to trigger auth challenge-response,
// then returns the final URI and headers that Clair can use to fetch the layer.
func buildLayerURI(ctx context.Context, httpClient *http.Client, ref name.Reference, layerDigest string) (string, map[string][]string, error) {
	repo := ref.Context()
	registryURL := url.URL{
		Scheme: repo.Scheme(),
		Host:   repo.RegistryStr(),
	}

	imgPath := strings.TrimPrefix(repo.RepositoryStr(), repo.RegistryStr())
	blobPath := path.Join("/", "v2", imgPath, "blobs", layerDigest)
	registryURL.Path = blobPath

	// Send a partial-content request to trigger auth and get final URI
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL.String(), nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Range", "bytes=0-0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("probing layer: %w", err)
	}
	defer resp.Body.Close()

	// The actual request (after redirects) has the auth headers we need
	finalReq := resp.Request
	headers := make(map[string][]string)
	for k, v := range finalReq.Header {
		// Copy auth-related headers, skip noise
		if k == "Authorization" || k == "Accept" {
			headers[k] = v
		}
	}

	// Remove Range header — we want Clair to fetch the full layer
	// Use the final URL (after any redirects)
	return finalReq.URL.String(), headers, nil
}
