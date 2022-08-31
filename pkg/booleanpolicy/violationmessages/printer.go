package violationmessages

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

type violationPrinter struct {
	required       set.StringSet // These fields must all be in the result, and must be valid search tags
	printerFuncKey string
}

var (
	policyFieldsToPrinters = map[string][]violationPrinter{
		fieldnames.AddCaps:                      {{required: set.NewStringSet(search.AddCapabilities.String()), printerFuncKey: printer.AddCapabilityKey}},
		fieldnames.AllowPrivilegeEscalation:     {{required: set.NewStringSet(search.AllowPrivilegeEscalation.String()), printerFuncKey: printer.AllowPrivilegeEscalationKey}},
		fieldnames.AppArmorProfile:              {{required: set.NewStringSet(search.AppArmorProfile.String()), printerFuncKey: printer.AppArmorProfileKey}},
		fieldnames.AutomountServiceAccountToken: {{required: set.NewStringSet(search.AutomountServiceAccountToken.String()), printerFuncKey: printer.AutomountServiceAccountTokenKey}},
		fieldnames.CVE:                          {{required: set.NewStringSet(search.CVE.String()), printerFuncKey: printer.CveKey}},
		fieldnames.CVSS:                         {{required: set.NewStringSet(search.CVE.String()), printerFuncKey: printer.CveKey}},
		fieldnames.ContainerCPULimit:            {{required: set.NewStringSet(search.CPUCoresLimit.String()), printerFuncKey: printer.ResourceKey}},
		fieldnames.ContainerCPURequest:          {{required: set.NewStringSet(search.CPUCoresRequest.String()), printerFuncKey: printer.ResourceKey}},
		fieldnames.ContainerMemLimit:            {{required: set.NewStringSet(search.MemoryLimit.String()), printerFuncKey: printer.ResourceKey}},
		fieldnames.ContainerMemRequest:          {{required: set.NewStringSet(search.MemoryRequest.String()), printerFuncKey: printer.ResourceKey}},
		fieldnames.ContainerName:                {{required: set.NewStringSet(search.ContainerName.String()), printerFuncKey: printer.ContainerNameKey}},
		fieldnames.DisallowedAnnotation:         {{required: set.NewStringSet(search.Annotation.String()), printerFuncKey: printer.DisallowedAnnotationKey}},
		fieldnames.DisallowedImageLabel:         {{required: set.NewStringSet(search.ImageLabel.String()), printerFuncKey: printer.DisallowedImageLabelKey}},
		fieldnames.DockerfileLine:               {{required: set.NewStringSet(augmentedobjs.DockerfileLineCustomTag), printerFuncKey: printer.LineKey}},
		fieldnames.DropCaps:                     {{required: set.NewStringSet(search.DropCapabilities.String()), printerFuncKey: printer.DropCapabilityKey}},
		fieldnames.EnvironmentVariable:          {{required: set.NewStringSet(augmentedobjs.EnvironmentVarCustomTag), printerFuncKey: printer.EnvKey}},
		fieldnames.ExposedNodePort:              {{required: set.NewStringSet(search.ExposedNodePort.String()), printerFuncKey: printer.NodePortKey}},
		fieldnames.ExposedPort:                  {{required: set.NewStringSet(search.Port.String()), printerFuncKey: printer.PortKey}},
		fieldnames.FixedBy:                      {{required: set.NewStringSet(search.CVE.String()), printerFuncKey: printer.CveKey}},
		fieldnames.HostIPC:                      {{required: set.NewStringSet(search.HostIPC.String()), printerFuncKey: printer.HostIPCKey}},
		fieldnames.HostNetwork:                  {{required: set.NewStringSet(search.HostNetwork.String()), printerFuncKey: printer.HostNetworkKey}},
		fieldnames.HostPID:                      {{required: set.NewStringSet(search.HostPID.String()), printerFuncKey: printer.HostPIDKey}},
		fieldnames.ImageAge:                     {{required: set.NewStringSet(search.ImageCreatedTime.String()), printerFuncKey: printer.ImageAgeKey}},
		fieldnames.ImageComponent:               {{required: set.NewStringSet(augmentedobjs.ComponentAndVersionCustomTag), printerFuncKey: printer.ComponentKey}},
		fieldnames.ImageOS:                      {{required: set.NewStringSet(search.ImageOS.String()), printerFuncKey: printer.ImageOSKey}},
		fieldnames.ImageRegistry:                {{required: set.StringSet{}, printerFuncKey: printer.ImageDetailsKey}},
		fieldnames.ImageRemote:                  {{required: set.StringSet{}, printerFuncKey: printer.ImageDetailsKey}},
		fieldnames.ImageScanAge:                 {{required: set.NewStringSet(search.ImageScanTime.String()), printerFuncKey: printer.ImageScanAgeKey}},
		fieldnames.ImageTag:                     {{required: set.StringSet{}, printerFuncKey: printer.ImageDetailsKey}},
		fieldnames.ImageUser:                    {{required: set.StringSet{}, printerFuncKey: printer.ImageUserKey}},
		fieldnames.ImageSignatureVerifiedBy:     {{required: set.NewStringSet(augmentedobjs.ImageSignatureVerifiedCustomTag), printerFuncKey: printer.ImageSignatureVerifiedKey}},
		fieldnames.LivenessProbeDefined:         {{required: set.NewStringSet(search.LivenessProbeDefined.String()), printerFuncKey: printer.LivenessProbeDefinedKey}},
		fieldnames.MinimumRBACPermissions:       {{required: set.NewStringSet(search.ServiceAccountPermissionLevel.String()), printerFuncKey: printer.RbacKey}},
		fieldnames.HasIngressNetworkPolicy:      {{required: set.NewStringSet(augmentedobjs.HasIngressPolicyCustomTag), printerFuncKey: printer.HasIngressNetworkPolicyKey}},
		fieldnames.HasEgressNetworkPolicy:       {{required: set.NewStringSet(augmentedobjs.HasEgressPolicyCustomTag), printerFuncKey: printer.HasEgressNetworkPolicyKey}},
		fieldnames.MountPropagation:             {{required: set.NewStringSet(search.MountPropagation.String()), printerFuncKey: printer.VolumeKey}},
		fieldnames.Namespace:                    {{required: set.NewStringSet(search.Namespace.String()), printerFuncKey: printer.NamespaceKey}},
		fieldnames.PortExposure:                 {{required: set.NewStringSet(search.ExposureLevel.String()), printerFuncKey: printer.PortExposureKey}},
		fieldnames.PrivilegedContainer:          {{required: set.NewStringSet(search.Privileged.String()), printerFuncKey: printer.PrivilegedKey}},
		fieldnames.ExposedPortProtocol:          {{required: set.NewStringSet(search.Port.String()), printerFuncKey: printer.PortKey}},
		fieldnames.ReadOnlyRootFS:               {{required: set.NewStringSet(search.ReadOnlyRootFilesystem.String()), printerFuncKey: printer.ReadOnlyRootFSKey}},
		fieldnames.Replicas:                     {{required: set.NewStringSet(search.Replicas.String()), printerFuncKey: printer.ReplicasKey}},
		fieldnames.ReadinessProbeDefined:        {{required: set.NewStringSet(search.ReadinessProbeDefined.String()), printerFuncKey: printer.ReadinessProbeDefinedKey}},
		fieldnames.RequiredAnnotation:           {{required: set.NewStringSet(search.Annotation.String()), printerFuncKey: printer.RequiredAnnotationKey}},
		fieldnames.RequiredImageLabel:           {{required: set.NewStringSet(search.ImageLabel.String()), printerFuncKey: printer.RequiredImageLabelKey}},
		fieldnames.RequiredLabel:                {{required: set.NewStringSet(search.DeploymentLabel.String()), printerFuncKey: printer.RequiredLabelKey}},
		fieldnames.RuntimeClass:                 {{required: set.NewStringSet(augmentedobjs.RuntimeClassCustomTag), printerFuncKey: printer.RuntimeClassKey}},
		fieldnames.SeccompProfileType:           {{required: set.NewStringSet(search.SeccompProfileType.String()), printerFuncKey: printer.SeccompProfileTypeKey}},
		fieldnames.ServiceAccount:               {{required: set.NewStringSet(search.ServiceAccountName.String()), printerFuncKey: printer.ServiceAccountKey}},
		fieldnames.Severity:                     {{required: set.NewStringSet(search.Severity.String()), printerFuncKey: printer.CveKey}},
		fieldnames.UnscannedImage:               {{required: set.NewStringSet(augmentedobjs.ImageScanCustomTag), printerFuncKey: printer.ImageScanKey}},
		fieldnames.VolumeDestination:            {{required: set.NewStringSet(search.VolumeName.String()), printerFuncKey: printer.VolumeKey}},
		fieldnames.VolumeName:                   {{required: set.NewStringSet(search.VolumeName.String()), printerFuncKey: printer.VolumeKey}},
		fieldnames.VolumeSource:                 {{required: set.NewStringSet(search.VolumeName.String()), printerFuncKey: printer.VolumeKey}},
		fieldnames.VolumeType:                   {{required: set.NewStringSet(search.VolumeName.String()), printerFuncKey: printer.VolumeKey}},
		fieldnames.WritableHostMount:            {{required: set.NewStringSet(search.VolumeName.String()), printerFuncKey: printer.VolumeKey}},
		fieldnames.WritableMountedVolume:        {{required: set.NewStringSet(search.VolumeName.String()), printerFuncKey: printer.VolumeKey}},
	}

	// runtime policy fields
	requiredProcessFields = set.NewFrozenStringSet(search.ProcessName.String(), search.ProcessAncestor.String(),
		search.ProcessUID.String(), search.ProcessArguments.String(), augmentedobjs.NotInProcessBaselineCustomTag)
	requiredKubeEventFields     = set.NewFrozenStringSet(augmentedobjs.KubernetesAPIVerbCustomTag, augmentedobjs.KubernetesResourceCustomTag)
	requiredNetworkFlowFields   = set.NewFrozenStringSet(augmentedobjs.NotInNetworkBaselineCustomTag)
	requiredNetworkPolicyFields = set.NewFrozenStringSet(augmentedobjs.HasEgressPolicyCustomTag, augmentedobjs.HasIngressPolicyCustomTag)
)

func containsAllRequiredFields(fieldMap map[string][]string, required set.StringSet) bool {
	for field := range required {
		if _, ok := fieldMap[field]; !ok {
			return false
		}
	}
	return true
}

func lookupViolationPrinters(section *storage.PolicySection, fieldMap map[string][]string) []printer.Func {
	keys := set.NewStringSet()
	for _, group := range section.GetPolicyGroups() {
		if printerMD, ok := policyFieldsToPrinters[group.GetFieldName()]; ok {
			for _, p := range printerMD {
				if containsAllRequiredFields(fieldMap, p.required) {
					keys.Add(p.printerFuncKey)
				}
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	return printer.GetFuncs(keys)
}

func checkForProcessViolation(result *evaluator.Result) bool {
	for _, fieldMap := range result.Matches {
		for k := range fieldMap {
			if requiredProcessFields.Contains(k) {
				return true
			}
		}
	}
	return false
}

func checkForKubeEventViolation(result *evaluator.Result) bool {
	for _, fieldMap := range result.Matches {
		for k := range fieldMap {
			if requiredKubeEventFields.Contains(k) {
				return true
			}
		}
	}
	return false
}

func checkForNetworkFlowViolation(result *evaluator.Result) bool {
	for _, fieldMap := range result.Matches {
		for k := range fieldMap {
			if requiredNetworkFlowFields.Contains(k) {
				return true
			}
		}
	}
	return false
}

func checkForNetworkPolicyViolation(result *evaluator.Result) bool {
	for _, fieldMap := range result.Matches {
		for k := range fieldMap {
			if requiredNetworkPolicyFields.Contains(k) {
				return true
			}
		}
	}
	return false
}

// Render creates violation messages based on evaluation results
func Render(
	section *storage.PolicySection,
	result *evaluator.Result,
	indicator *storage.ProcessIndicator,
	kubeEvent *storage.KubernetesEvent,
	networkFlow *augmentedobjs.NetworkFlowDetails,
	networkPolicy *augmentedobjs.NetworkPoliciesApplied,
) ([]*storage.Alert_Violation, bool, bool, bool, bool, error) {
	errorList := errorhelpers.NewErrorList("violation printer")
	messages := set.NewStringSet()
	for _, fieldMap := range result.Matches {
		printers := lookupViolationPrinters(section, fieldMap)
		if len(printers) == 0 {
			continue
		}
		for _, printerFunc := range printers {
			messagesForResult, err := printerFunc(fieldMap)
			if err != nil {
				errorList.AddError(err)
				continue
			}
			messages.AddAll(messagesForResult...)
		}
	}

	isProcessViolation := indicator != nil && checkForProcessViolation(result)
	isKubeOrAuditEventViolation := kubeEvent != nil && checkForKubeEventViolation(result)
	isNetworkFlowViolation := networkFlow != nil && checkForNetworkFlowViolation(result)
	isNetworkPolicyViolation := networkPolicy != nil && checkForNetworkPolicyViolation(result)
	if len(messages) == 0 && !isProcessViolation && !isKubeOrAuditEventViolation && !isNetworkFlowViolation {
		errorList.AddError(errors.New("missing messages"))
	}

	alertType := storage.Alert_Violation_GENERIC
	if isNetworkPolicyViolation {
		alertType = storage.Alert_Violation_NETWORK_POLICY
	}
	alertViolations := make([]*storage.Alert_Violation, 0, len(messages))
	// Sort messages for consistency in output. This is important because we
	// depend on these messages being equal when deduping updates to alerts.
	for _, message := range messages.AsSortedSlice(func(i, j string) bool {
		return i < j
	}) {
		alertViolations = append(alertViolations, &storage.Alert_Violation{
			Message: message,
			Type:    alertType,
		})
	}
	return alertViolations, isProcessViolation, isKubeOrAuditEventViolation, isNetworkFlowViolation, isNetworkPolicyViolation, errorList.ToError()
}
