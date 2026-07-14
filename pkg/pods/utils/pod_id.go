package utils

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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

var errInvalidPodID = fmt.Errorf("invalid Pod ID")

// ParsePodID takes a string and returns the parsed pod ID, or an error.
func ParsePodID(str string) (PodID, error) {
	atIdx := strings.IndexByte(str, '@')
	if atIdx < 0 || atIdx == len(str)-1 {
		return PodID{}, errInvalidPodID
	}
	uid := str[atIdx+1:]
	dotIdx := strings.LastIndexByte(str[:atIdx], '.')
	if dotIdx < 0 {
		return PodID{}, errInvalidPodID
	}
	name := str[:dotIdx]
	namespace := str[dotIdx+1 : atIdx]
	if !isValidDNSSubdomain(name) || !isValidDNSLabel(namespace) || !isValidUID(uid) {
		return PodID{}, errInvalidPodID
	}
	return PodID{
		Name:      name,
		Namespace: namespace,
		UID:       types.UID(uid),
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

func isLowerAlphaNum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func isValidDNSSubdomain(s string) bool {
	if len(s) == 0 || !isLowerAlphaNum(s[0]) || !isLowerAlphaNum(s[len(s)-1]) {
		return false
	}
	for i := 1; i < len(s)-1; i++ {
		c := s[i]
		if !isLowerAlphaNum(c) && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

func isValidDNSLabel(s string) bool {
	if len(s) == 0 || !isLowerAlphaNum(s[0]) || !isLowerAlphaNum(s[len(s)-1]) {
		return false
	}
	for i := 1; i < len(s)-1; i++ {
		c := s[i]
		if !isLowerAlphaNum(c) && c != '-' {
			return false
		}
	}
	return true
}

func isValidUID(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') || c == '-') {
			return false
		}
	}
	return true
}
