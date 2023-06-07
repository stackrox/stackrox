package common

import "strings"

const (
	// CentralPVCObsoleteAnnotation represents Central PVC has been obsoleted
	CentralPVCObsoleteAnnotation = "platform.stackrox.io/obsolete-central-pvc"
)

// ObsoletePVC determines if we should obsolete PVC
func ObsoletePVC(annotations map[string]string) bool {
	return strings.EqualFold(strings.TrimSpace(annotations[CentralPVCObsoleteAnnotation]), "true")
}
