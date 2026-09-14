package operatorbundle

import (
	"github.com/stackrox/rox/generated/storage"
)

// CVE is a single vulnerability finding on an image, reduced to the fields needed for
// diffing and reporting.
type CVE struct {
	ID       string
	Severity storage.VulnerabilitySeverity
	CVSS     float32
	// FixedBy is the component version that resolves the CVE; empty means not fixable.
	FixedBy string
}

// IsFixable reports whether a fix is available for the CVE.
func (c CVE) IsFixable() bool {
	return c.FixedBy != ""
}

// ImageCVEs is the set of CVEs for a single image, identified by its repository name
// (registry + repository, without tag or digest) and its digest.
type ImageCVEs struct {
	// Repository is the tag/digest-stripped image reference, e.g.
	// "registry.redhat.io/albo/aws-load-balancer-controller-rhel9". It is the key used to
	// pair installed and candidate images during diffing.
	Repository string
	// Reference is the full reference the scan was obtained from (repository@digest), kept
	// for reporting.
	Reference string
	Digest    string
	CVEs      []CVE
}

// RelatedImage is an image that belongs to an operator bundle, as returned by the catalog
// find_operator_bundles.related_images field.
type RelatedImage struct {
	// Image is the image reference as returned by the catalog, e.g.
	// "registry.redhat.io/albo/aws-load-balancer-controller-rhel9".
	Image  string
	Digest string
}

// Bundle is a single operator bundle version from the catalog.
type Bundle struct {
	Package         string
	ChannelName     string
	Version         string
	VersionOriginal string
	CSVName         string
	CSVDisplayName  string
	// CreationDate is the catalog creation_date (RFC3339 / date string), used as the lower
	// bound when querying for candidate bundles.
	CreationDate  string
	RelatedImages []RelatedImage
}

// ImagePairStatus describes how an image participates in the diff between the installed and
// the update-candidate bundle.
type ImagePairStatus string

const (
	// ImagePaired means the repository exists in both bundles and CVEs were diffed.
	ImagePaired ImagePairStatus = "PAIRED"
	// ImageAdded means the repository exists only in the update-candidate bundle.
	ImageAdded ImagePairStatus = "ADDED"
	// ImageRemoved means the repository exists only in the installed bundle.
	ImageRemoved ImagePairStatus = "REMOVED"
)

// ImageDiff is the per-image CVE diff between the installed and update-candidate bundle for
// a single repository.
type ImageDiff struct {
	Repository      string
	Status          ImagePairStatus
	InstalledDigest string
	CandidateDigest string
	// Fixed CVEs are present in the installed image but not in the candidate image.
	Fixed []CVE
	// StillActive CVEs are present in both images.
	StillActive []CVE
	// New CVEs are present in the candidate image but not in the installed image.
	New []CVE
}

// BundleDiffReport is the result of Advisor.Advise for a single installed bundle and its
// selected update candidate.
type BundleDiffReport struct {
	Package         string
	ChannelName     string
	InstalledBundle Bundle
	// UpdateCandidate is the bundle selected as the update target. It is nil when no newer
	// patch release exists within the installed bundle's major.minor (see NoUpdateReason).
	UpdateCandidate *Bundle
	// NoUpdateReason is set when UpdateCandidate is nil, explaining why.
	NoUpdateReason string
	// TriggeringDigests are the input image digests that resolved to this installed bundle.
	TriggeringDigests []string
	ImageDiffs        []ImageDiff
}

// HasUpdate reports whether an update candidate was found.
func (r BundleDiffReport) HasUpdate() bool {
	return r.UpdateCandidate != nil
}
