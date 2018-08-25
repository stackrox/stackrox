package resources

import (
	"fmt"
	"regexp"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	dns1123Regex = `[a-z0-9](?:[-a-z0-9]*[a-z0-9])?`
	uidRegex     = `[[:xdigit:]-]+`
)

var (
	podIDRegex = regexp.MustCompile(`^(` + dns1123Regex + `)\.(` + dns1123Regex + `)@(` + uidRegex + `)$`)
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

// getPodID returns a pod ID for the given pod object.
func getPodID(pod v1.Pod) PodID {
	return PodID{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		UID:       pod.UID,
	}
}
