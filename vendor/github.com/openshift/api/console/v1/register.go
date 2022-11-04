package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	GroupName     = "console.openshift.io"
	GroupVersion  = schema.GroupVersion{Group: GroupName, Version: "v1"}
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes, corev1.AddToScheme)
	// Install is a function which adds this version to a scheme
	Install = schemeBuilder.AddToScheme

	// SchemeGroupVersion generated code relies on this name
	// Deprecated
	SchemeGroupVersion = GroupVersion
	// AddToScheme exists solely to keep the old generators creating valid code
	// DEPRECATED
	AddToScheme = schemeBuilder.AddToScheme
)

// Resource generated code relies on this being here, but it logically belongs to the group
// DEPRECATED
func Resource(resource string) schema.GroupResource {
	return schema.GroupResource{Group: GroupName, Resource: resource}
}

// addKnownTypes adds types to API group
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&ConsoleLink{},
		&ConsoleLinkList{},
		&ConsoleCLIDownload{},
		&ConsoleCLIDownloadList{},
		&ConsoleNotification{},
		&ConsoleNotificationList{},
		&ConsoleExternalLogLink{},
		&ConsoleExternalLogLinkList{},
		&ConsoleYAMLSample{},
		&ConsoleYAMLSampleList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
