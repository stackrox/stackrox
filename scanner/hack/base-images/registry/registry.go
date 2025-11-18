package registry

import (
	"crypto/sha1"
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"localhost/jvdm/image-puller-poc/jsoncache"
	"time"
)

type InspectPayload struct {
	Layers  []string   `json:"Layers"`
	Created *time.Time `json:"Created,omitempty"`
}

// Client is a high-level interface to a container registry.
type Client struct {
	cache    *jsoncache.JSONCache
	keychain authn.Keychain
	platform v1.Platform
}

func NewClient(cache *jsoncache.JSONCache, keychain authn.Keychain, platform v1.Platform) *Client {
	return &Client{cache: cache, keychain: keychain, platform: platform}
}

func (rc *Client) authID(authfile string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(authfile))
	return fmt.Sprintf("%x", h.Sum(nil))[:12]
}

func (rc *Client) cacheKey(ref, platform, authfile string) string {
	return fmt.Sprintf("inspect|ref=%s|plat=%s|auth=%s", ref, platform, rc.authID(authfile))
}

// InspectRef fetches Layers (compressed digests) and Created time for a reference.
// If cache==true, the result is read/written to the json cache.
func (rc *Client) InspectRef(ref string, dockerConfigPath string, useCache bool) (InspectPayload, error) {
	key := rc.cacheKey(ref, fmt.Sprintf("%s/%s", rc.platform.OS, rc.platform.Architecture), dockerConfigPath)
	var payload InspectPayload

	if useCache {
		if ok, err := rc.cache.Get(key, &payload); err != nil {
			return InspectPayload{}, err
		} else if ok {
			return payload, nil
		}
	}

	nref, err := name.ParseReference(ref)
	if err != nil {
		return InspectPayload{}, err
	}

	img, err := remote.Image(
		nref,
		remote.WithAuthFromKeychain(rc.keychain),
		remote.WithPlatform(rc.platform),
	)
	if err != nil {
		return InspectPayload{}, err
	}

	m, err := img.Manifest()
	if err != nil {
		return InspectPayload{}, err
	}
	layers := make([]string, 0, len(m.Layers))
	for _, d := range m.Layers {
		layers = append(layers, d.Digest.String())
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return InspectPayload{}, err
	}

	var created *time.Time
	if cfg != nil && !cfg.Created.IsZero() {
		t := cfg.Created.UTC()
		created = &t
	}

	payload = InspectPayload{Layers: layers, Created: created}
	if useCache {
		if err := rc.cache.Set(key, payload); err != nil {
			return InspectPayload{}, err
		}
	}
	return payload, nil
}

// ListTags returns all tags for a repository path like "docker.io/library/ubuntu".
func (rc *Client) ListTags(repoPath string) ([]string, error) {
	tags, err := crane.ListTags(repoPath, crane.WithAuthFromKeychain(rc.keychain))
	if err != nil {
		return nil, err
	}
	return tags, nil
}
