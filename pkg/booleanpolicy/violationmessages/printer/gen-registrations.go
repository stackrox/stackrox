// Code generated by violationprinter codegen. DO NOT EDIT.

package printer

const (
	AddCapabilityKey                = "addCapability"
	AllowPrivilegeEscalationKey     = "allowPrivilegeEscalation"
	AppArmorProfileKey              = "appArmorProfile"
	AutomountServiceAccountTokenKey = "automountServiceAccountToken"
	ComponentKey                    = "component"
	ContainerNameKey                = "containerName"
	CveKey                          = "cve"
	DisallowedAnnotationKey         = "disallowedAnnotation"
	DisallowedImageLabelKey         = "disallowedImageLabel"
	DropCapabilityKey               = "dropCapability"
	EnvKey                          = "env"
	HasEgressNetworkPolicyKey       = "hasEgressNetworkPolicy"
	HasIngressNetworkPolicyKey      = "hasIngressNetworkPolicy"
	HostIPCKey                      = "hostIPC"
	HostNetworkKey                  = "hostNetwork"
	HostPIDKey                      = "hostPID"
	ImageAgeKey                     = "imageAge"
	ImageDetailsKey                 = "imageDetails"
	ImageOSKey                      = "imageOS"
	ImageScanKey                    = "imageScan"
	ImageScanAgeKey                 = "imageScanAge"
	ImageSignatureVerifiedKey       = "imageSignatureVerified"
	ImageUserKey                    = "imageUser"
	LineKey                         = "line"
	LivenessProbeDefinedKey         = "livenessProbeDefined"
	NamespaceKey                    = "namespace"
	NodePortKey                     = "nodePort"
	PortKey                         = "port"
	PortExposureKey                 = "portExposure"
	PrivilegedKey                   = "privileged"
	ProcessBaselineKey              = "processBaseline"
	RbacKey                         = "rbac"
	ReadOnlyRootFSKey               = "readOnlyRootFS"
	ReadinessProbeDefinedKey        = "readinessProbeDefined"
	ReplicasKey                     = "replicas"
	RequiredAnnotationKey           = "requiredAnnotation"
	RequiredImageLabelKey           = "requiredImageLabel"
	RequiredLabelKey                = "requiredLabel"
	ResourceKey                     = "resource"
	RuntimeClassKey                 = "runtimeClass"
	SeccompProfileTypeKey           = "seccompProfileType"
	ServiceAccountKey               = "serviceAccount"
	VolumeKey                       = "volume"
)

func init() {
	registerFunc(AddCapabilityKey, addCapabilityPrinter)
	registerFunc(AllowPrivilegeEscalationKey, allowPrivilegeEscalationPrinter)
	registerFunc(AppArmorProfileKey, appArmorProfilePrinter)
	registerFunc(AutomountServiceAccountTokenKey, automountServiceAccountTokenPrinter)
	registerFunc(ComponentKey, componentPrinter)
	registerFunc(ContainerNameKey, containerNamePrinter)
	registerFunc(CveKey, cvePrinter)
	registerFunc(DisallowedAnnotationKey, disallowedAnnotationPrinter)
	registerFunc(DisallowedImageLabelKey, disallowedImageLabelPrinter)
	registerFunc(DropCapabilityKey, dropCapabilityPrinter)
	registerFunc(EnvKey, envPrinter)
	registerFunc(HasEgressNetworkPolicyKey, hasEgressNetworkPolicyPrinter)
	registerFunc(HasIngressNetworkPolicyKey, hasIngressNetworkPolicyPrinter)
	registerFunc(HostIPCKey, hostIPCPrinter)
	registerFunc(HostNetworkKey, hostNetworkPrinter)
	registerFunc(HostPIDKey, hostPIDPrinter)
	registerFunc(ImageAgeKey, imageAgePrinter)
	registerFunc(ImageDetailsKey, imageDetailsPrinter)
	registerFunc(ImageOSKey, imageOSPrinter)
	registerFunc(ImageScanKey, imageScanPrinter)
	registerFunc(ImageScanAgeKey, imageScanAgePrinter)
	registerFunc(ImageSignatureVerifiedKey, imageSignatureVerifiedPrinter)
	registerFunc(ImageUserKey, imageUserPrinter)
	registerFunc(LineKey, linePrinter)
	registerFunc(LivenessProbeDefinedKey, livenessProbeDefinedPrinter)
	registerFunc(NamespaceKey, namespacePrinter)
	registerFunc(NodePortKey, nodePortPrinter)
	registerFunc(PortKey, portPrinter)
	registerFunc(PortExposureKey, portExposurePrinter)
	registerFunc(PrivilegedKey, privilegedPrinter)
	registerFunc(ProcessBaselineKey, processBaselinePrinter)
	registerFunc(RbacKey, rbacPrinter)
	registerFunc(ReadOnlyRootFSKey, readOnlyRootFSPrinter)
	registerFunc(ReadinessProbeDefinedKey, readinessProbeDefinedPrinter)
	registerFunc(ReplicasKey, replicasPrinter)
	registerFunc(RequiredAnnotationKey, requiredAnnotationPrinter)
	registerFunc(RequiredImageLabelKey, requiredImageLabelPrinter)
	registerFunc(RequiredLabelKey, requiredLabelPrinter)
	registerFunc(ResourceKey, resourcePrinter)
	registerFunc(RuntimeClassKey, runtimeClassPrinter)
	registerFunc(SeccompProfileTypeKey, seccompProfileTypePrinter)
	registerFunc(ServiceAccountKey, serviceAccountPrinter)
	registerFunc(VolumeKey, volumePrinter)
}
