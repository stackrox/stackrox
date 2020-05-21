package utils

import (
	"fmt"
	"regexp"

	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// Resource name regexes based on https://kubernetes.io/docs/concepts/overview/working-with-objects/names/
	// and https://github.com/kubernetes/community/blob/master/contributors/design-proposals/architecture/identifiers.md
	dnsSubdomain1123Regex = `[a-z0-9](?:[-\.a-z0-9]*[a-z0-9])?`
	dnsLabel1123Regex     = `[a-z0-9](?:[-a-z0-9]*[a-z0-9])?`
	uidRegex              = `[[:xdigit:]-]+`
)

var (
	podIDRegex = regexp.MustCompile(`^(` + dnsSubdomain1123Regex + `)\.(` + dnsLabel1123Regex + `)@(` + uidRegex + `)$`)
)

// PodID allows uniquely identifying a pod instance.
type PodID struct {
	Name      string
	Namespace string
	UID       types.UID
}

// String returns the string representation for the given Pod ID.
func (p PodID) String() string {
	if p.IsEmpty() {
		return ""
	}
	return fmt.Sprintf("%s.%s@%s", p.Name, p.Namespace, p.UID)
}

// IsEmpty checks whether this pod ID is the empty pod ID.
func (p PodID) IsEmpty() bool {
	return p.Name == "" && p.Namespace == "" && p.UID == ""
}

// ParsePodID takes a string and returns the parsed pod ID, or an error.
func ParsePodID(str string) (PodID, error) {
	matches := podIDRegex.FindStringSubmatch(str)
	if len(matches) != 4 {
		return PodID{}, fmt.Errorf("string %q is not a valid Pod ID; regex used for validation is %q", str, podIDRegex)
	}
	return PodID{
		Name:      matches[1],
		Namespace: matches[2],
		UID:       types.UID(matches[3]),
	}, nil
}

// GetPodIDFromV1Pod returns a pod ID for the given pod object.
func GetPodIDFromV1Pod(pod *v1.Pod) PodID {
	return PodID{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		UID:       pod.UID,
	}
}

// GetPodIDFromStoragePod returns a pod ID for the given pod object.
func GetPodIDFromStoragePod(pod *storage.Pod) PodID {
	return PodID{
		Name:      pod.GetName(),
		Namespace: pod.GetNamespace(),
		UID:       types.UID(pod.GetId()),
	}
}
