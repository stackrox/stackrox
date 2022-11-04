package registry

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/docker/distribution/manifest/ocischema"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	digest "github.com/opencontainers/go-digest"
	ociSpec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	MediaTypeManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
)

type Platform struct {
	Architecture string `json:"architecture"`
	OS           string `json:"os"`
}

type Manifest struct {
	MediaType string   `json:"mediaType"`
	Digest    string   `json:"digest"`
	Platform  Platform `json:"platform"`
}

type ManifestList struct {
	canonical []byte
	Manifests []Manifest `json:"manifests"`
}

func (m *ManifestList) Canonical() []byte {
	if m == nil {
		return nil
	}
	return m.canonical
}

func (registry *Registry) Manifest(repository, reference string) (*manifestV1.SignedManifest, error) {
	return registry.v1Manifest(repository, reference, manifestV1.MediaTypeManifest)
}

func (registry *Registry) SignedManifest(repository, reference string) (*manifestV1.SignedManifest, error) {
	return registry.v1Manifest(repository, reference, manifestV1.MediaTypeSignedManifest)
}

func (registry *Registry) ManifestList(repository, reference string) (*ManifestList, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", MediaTypeManifestList)
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	manifestList := &ManifestList{
		canonical: body,
	}
	if err := json.Unmarshal(body, &manifestList); err != nil {
		return nil, err
	}
	return manifestList, nil
}

func (registry *Registry) v1Manifest(repository, reference string, mediaType string) (*manifestV1.SignedManifest, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", mediaType)
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	signedManifest := &manifestV1.SignedManifest{}
	err = signedManifest.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}

	return signedManifest, nil
}

func (registry *Registry) ManifestV2(repository, reference string) (*manifestV2.DeserializedManifest, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", manifestV2.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	deserialized := &manifestV2.DeserializedManifest{}
	err = deserialized.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	return deserialized, nil
}

func (registry *Registry) ManifestOCI(repository, reference string) (*ocischema.DeserializedManifest, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.get url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", ociSpec.MediaTypeImageManifest)
	resp, err := registry.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	deserialized := &ocischema.DeserializedManifest{}
	err = deserialized.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}
	return deserialized, nil
}


func (registry *Registry) ManifestDigest(repository, reference string) (digest.Digest, string, error) {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.head url=%s repository=%s reference=%s", url, repository, reference)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Add("Accept", manifestV2.MediaTypeManifest)
	req.Header.Add("Accept", manifestV1.MediaTypeManifest)
	req.Header.Add("Accept", manifestV1.MediaTypeSignedManifest)
	req.Header.Add("Accept", MediaTypeManifestList)
	req.Header.Add("Accept", ociSpec.MediaTypeImageManifest)

	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", "", err
	}

	contentType := resp.Header.Get("Content-Type")
	d, err := digest.Parse(resp.Header.Get("Docker-Content-Digest"))
	return d, contentType, err
}

func (registry *Registry) DeleteManifest(repository string, digest digest.Digest) error {
	url := registry.url("/v2/%s/manifests/%s", repository, digest)
	registry.Logf("registry.manifest.delete url=%s repository=%s reference=%s", url, repository, digest)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	return nil
}

func (registry *Registry) PutManifest(repository, reference string, signedManifest *manifestV1.SignedManifest) error {
	url := registry.url("/v2/%s/manifests/%s", repository, reference)
	registry.Logf("registry.manifest.put url=%s repository=%s reference=%s", url, repository, reference)

	body, err := signedManifest.MarshalJSON()
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(body)
	req, err := http.NewRequest("PUT", url, buffer)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", manifestV1.MediaTypeManifest)
	resp, err := registry.Client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	return err
}
