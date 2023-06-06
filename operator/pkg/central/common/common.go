package common

import "strings"

const (
	// CentralPVCObsoleteAnnotation represents Central PVC has been obsoleted
	CentralPVCObsoleteAnnotation = "platform.stackrox.io/obsolete-central-pvc"
)

// ObsoletePVC determines if we should obsolete PVC
func ObsoletePVC(annotations map[string]string) (obsoletePVC bool) {
	if value, ok := annotations[CentralPVCObsoleteAnnotation]; ok {
		obsoletePVC = strings.EqualFold("true", strings.TrimSpace(value))
	}
	return
}
