package nvd

// Project represents an individual project for which there exists an NVD CVE
// data feed.
type Project int

const (
	// Kubernetes represents the Kubernetes project.
	Kubernetes Project = iota + 1

	// Istio represents the Istio project.
	Istio

	// Openshift represents the Openshift project
	Openshift
)

func (p Project) String() string {
	switch p {
	case Istio:
		return "istio"
	case Kubernetes:
		return "kubernetes"
	case Openshift:
		return "openshift"
	default:
		return "unknown"
	}
}

func (p Project) Vendor() string {
	switch p {
	case Istio:
		return "istio"
	case Kubernetes:
		return "kubernetes"
	case Openshift:
		return "redhat"
	default:
		return "unknown"
	}
}

// Feed represents an individual NVD CVE data feed and the associated file
// paths.
type Feed struct {
	Name             string
	Description      string
	CVEFilename      string
	ChecksumFilename string
}

var (
	// Feeds represents the set of currently available NVD CVE data feeds.
	Feeds = map[Project]Feed{
		Istio: {
			Name:             "istio",
			Description:      "NVD CVE data for the Istio project",
			CVEFilename:      "istio/cve-list.json",
			ChecksumFilename: "istio/checksum",
		},
		Kubernetes: {
			Name:             "k8s",
			Description:      "NVD CVE data for the Kubernetes project",
			CVEFilename:      "k8s/cve-list.json",
			ChecksumFilename: "k8s/checksum",
		},
		Openshift: {
			Name:             "openshift",
			Description:      "NVD CVE data for the Openshift project",
			CVEFilename:      "openshift/cve-list.json",
			ChecksumFilename: "openshift/checksum",
		},
	}
)
