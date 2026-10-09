// Package ocprelease provides an operatorbundle.CatalogClient backed by OpenShift release
// payloads. The cluster's current release and its available updates are read from the
// ClusterVersion CR; each release's component image set is extracted from the release image's
// release-manifests/image-references file. This lets the existing operatorbundle.Advisor diff the
// CVEs of the component images an OCP upgrade would change, per image.
package ocprelease

import (
	"archive/tar"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	gcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/operatorbundle"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Package is the synthetic package name used for the OpenShift platform "bundle".
const Package = "openshift"

const imageReferencesPath = "release-manifests/image-references"

var clusterVersionGVR = schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "clusterversions"}

// Client implements operatorbundle.CatalogClient for OpenShift release payloads.
type Client struct {
	dyn  dynamic.Interface
	auth authn.Authenticator
}

// NewClient builds a Client from a dynamic Kubernetes client (to read ClusterVersion) and an
// authenticator for the release registry (quay.io/openshift-release-dev).
func NewClient(dyn dynamic.Interface, auth authn.Authenticator) *Client {
	if auth == nil {
		auth = authn.Anonymous
	}
	return &Client{dyn: dyn, auth: auth}
}

type release struct {
	version string
	image   string
}

// readClusterVersion returns the cluster's current release and its available updates.
func (c *Client) readClusterVersion(ctx context.Context) (release, []release, error) {
	cv, err := c.dyn.Resource(clusterVersionGVR).Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return release{}, nil, errors.Wrap(err, "reading ClusterVersion")
	}
	current := release{
		version: nestedString(cv, "status", "desired", "version"),
		image:   nestedString(cv, "status", "desired", "image"),
	}
	var available []release
	updates, _, _ := unstructured.NestedSlice(cv.Object, "status", "availableUpdates")
	for _, u := range updates {
		m, ok := u.(map[string]interface{})
		if !ok {
			continue
		}
		v, _ := m["version"].(string)
		img, _ := m["image"].(string)
		if v != "" && img != "" {
			available = append(available, release{version: v, image: img})
		}
	}
	return current, available, nil
}

// FindInstalledBundle returns the cluster's current OpenShift release as a bundle. The digest
// argument is ignored: the current release is determined from the ClusterVersion CR, not from the
// triggering image.
func (c *Client) FindInstalledBundle(ctx context.Context, _ string) (*operatorbundle.Bundle, error) {
	current, _, err := c.readClusterVersion(ctx)
	if err != nil {
		return nil, err
	}
	if current.version == "" || current.image == "" {
		return nil, nil
	}
	return c.bundleForRelease(ctx, current)
}

// FindCandidateBundles returns the cluster's available-update releases as bundles. The pkg,
// channel and sinceCreationDate arguments are unused; the advisor's SelectLatestPatch chooses the
// target by semantic version.
func (c *Client) FindCandidateBundles(ctx context.Context, _, _, _ string) ([]operatorbundle.Bundle, error) {
	_, available, err := c.readClusterVersion(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]operatorbundle.Bundle, 0, len(available))
	for _, r := range available {
		b, err := c.bundleForRelease(ctx, r)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, nil
}

func (c *Client) bundleForRelease(ctx context.Context, r release) (*operatorbundle.Bundle, error) {
	images, err := ExtractImageReferences(ctx, r.image, c.auth)
	if err != nil {
		return nil, errors.Wrapf(err, "extracting image references for release %s", r.version)
	}
	return &operatorbundle.Bundle{
		Package:         Package,
		Version:         r.version,
		VersionOriginal: r.version,
		CSVName:         Package + "@" + r.version,
		CSVDisplayName:  "OpenShift " + r.version,
		RelatedImages:   images,
	}, nil
}

// imageStream is the minimal shape of a release's release-manifests/image-references file.
type imageStream struct {
	Spec struct {
		Tags []struct {
			Name string `json:"name"`
			From struct {
				Name string `json:"name"`
			} `json:"from"`
		} `json:"tags"`
	} `json:"spec"`
}

// ExtractImageReferences pulls an OpenShift release image and returns its component images from the
// embedded release-manifests/image-references ImageStream, as operatorbundle RelatedImages keyed by
// component name. The release's components all share one repository, so the (repository, name) diff
// pairing reduces to the component name.
func ExtractImageReferences(ctx context.Context, releaseImage string, auth authn.Authenticator) ([]operatorbundle.RelatedImage, error) {
	img, err := crane.Pull(releaseImage,
		crane.WithContext(ctx),
		crane.WithAuth(auth),
		crane.WithPlatform(&gcrv1.Platform{OS: "linux", Architecture: "amd64"}),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "pulling release image %s", releaseImage)
	}

	pr, pw := io.Pipe()
	go func() { _ = pw.CloseWithError(crane.Export(img, pw)) }()
	defer func() { _ = pr.Close() }() // unblocks Export if we stop reading early

	tr := tar.NewReader(pr)
	var refsJSON []byte
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "reading release image filesystem")
		}
		if strings.TrimPrefix(h.Name, "./") == imageReferencesPath {
			refsJSON, err = io.ReadAll(tr)
			if err != nil {
				return nil, errors.Wrap(err, "reading image-references")
			}
			break
		}
	}
	if refsJSON == nil {
		return nil, errors.Errorf("%s not found in release image %s", imageReferencesPath, releaseImage)
	}
	return parseImageReferences(refsJSON)
}

// parseImageReferences converts a release's image-references ImageStream JSON into RelatedImages
// keyed by component name.
func parseImageReferences(refsJSON []byte) ([]operatorbundle.RelatedImage, error) {
	var is imageStream
	if err := json.Unmarshal(refsJSON, &is); err != nil {
		return nil, errors.Wrap(err, "parsing image-references")
	}
	out := make([]operatorbundle.RelatedImage, 0, len(is.Spec.Tags))
	for _, t := range is.Spec.Tags {
		out = append(out, operatorbundle.RelatedImage{
			Image:  t.From.Name,
			Name:   t.Name,
			Digest: digestFromRef(t.From.Name),
		})
	}
	return out, nil
}

func digestFromRef(ref string) string {
	if i := strings.Index(ref, "@"); i >= 0 {
		return ref[i+1:]
	}
	return ""
}

func nestedString(u *unstructured.Unstructured, fields ...string) string {
	s, _, _ := unstructured.NestedString(u.Object, fields...)
	return s
}
