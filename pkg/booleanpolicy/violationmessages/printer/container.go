package printer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
)

const (
	readOnlyRootFSTemplate = `Container {{if .ContainerName}}'{{.ContainerName}}'{{end}} 
	{{- if .ReadOnlyRootFS }} uses a read-only root filesystem{{else}} uses a read-write root filesystem{{end}}`
)

func readOnlyRootFSPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName  string
		ReadOnlyRootFS bool
	}

	r := resultFields{}
	var err error
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	readOnlyRootFS, err := getSingleValueFromFieldMap(search.ReadOnlyRootFilesystem.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.ReadOnlyRootFS, err = strconv.ParseBool(readOnlyRootFS); err != nil {
		return nil, err
	}
	return executeTemplate(readOnlyRootFSTemplate, r)
}

const (
	imageAgeTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' has image{{else}}Image was{{end}} created at {{.ImageCreationTime}} (UTC)`
)

func imageAgePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName     string
		ImageCreationTime string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.ImageCreationTime, err = getSingleValueFromFieldMap(search.ImageCreatedTime.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(imageAgeTemplate, r)
}

const (
	imageScanAgeTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' has image{{else}}Image was{{end}} last scanned at {{.ImageScanTime}} (UTC)`
)

func imageScanAgePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ImageScanTime string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.ImageScanTime, err = getSingleValueFromFieldMap(search.ImageScanTime.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(imageScanAgeTemplate, r)
}

const (
	imageOSTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' has image with{{else}}Image has{{end}} base OS '{{.ImageOS}}'`
)

func imageOSPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ImageOS       string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.ImageOS, err = getSingleValueFromFieldMap(search.ImageOS.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(imageOSTemplate, r)
}

const (
	imageDetailsTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' has image with{{else}}Image has{{end}} {{.ImageDetails}}`
)

func imageDetailsPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ImageDetails  string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var imageDetails []string
	if imageTag, err := getSingleValueFromFieldMap(search.ImageTag.String(), fieldMap); err == nil {
		imageDetails = append(imageDetails, fmt.Sprintf("tag '%s'", imageTag))
	}
	if imageRemote, err := getSingleValueFromFieldMap(search.ImageRemote.String(), fieldMap); err == nil {
		imageDetails = append(imageDetails, fmt.Sprintf("remote '%s'", imageRemote))
	}
	if imageRegistry, err := getSingleValueFromFieldMap(search.ImageRegistry.String(), fieldMap); err == nil {
		imageDetails = append(imageDetails, fmt.Sprintf("registry '%s'", imageRegistry))
	}
	// This is okay, it can happen if this fieldMap has values for other fields.
	if len(imageDetails) == 0 {
		return nil, nil
	}
	r.ImageDetails = StringSliceToSortedSentence(imageDetails)
	return executeTemplate(imageDetailsTemplate, r)
}

const (
	privilegedTemplate = `Container{{if .ContainerName}} '{{.ContainerName}}'{{end}} is{{if not .Privileged}} not{{end}} privileged`
)

// Render violation message for match against policyFieldsToPrinters privileged container
func privilegedPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		Privileged    bool
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	privileged, err := getSingleValueFromFieldMap(search.Privileged.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.Privileged, err = strconv.ParseBool(privileged); err != nil {
		return nil, err
	}
	return executeTemplate(privilegedTemplate, r)
}

const (
	imageUserTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' has image with{{else}}Image has{{end}} user '{{.ImageUser}}'`
)

func imageUserPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName string
		ImageUser     string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.ImageUser, err = getSingleValueFromFieldMap(search.ImageUser.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(imageUserTemplate, r)
}

// imageSignatureVerifiedPrinter returns a violation message explaining that the image signature is not verified by the
// specified signature integration. If other signature integrations did verify the image, they are listed in the
// message.
// Example outputs (in all cases a signature integration failed to verify the image and raises a violation):
//   - No other integrations verified the image:
//     "Image signature is not verified by the specified signature integration(s)."
//   - A different integration does verify the image:
//     "Image signature is not verified by the specified signature integration(s) (it is verified by other integration(s): io.stackrox.signatureintegration.3fee323b-da48-4fe2-8041-02e0740cc4f5)."
//   - Several different integrations verify the image:
//     "Image signature is not verified by the specified signature integration(s) (it is verified by other integration(s): io.stackrox.signatureintegration.3fee323b-da48-4fe2-8041-02e0740cc4f5 and io.stackrox.signatureintegration.a9ab4422-fa1d-4c99-a545-ea33ca57c8f8)."
func imageSignatureVerifiedPrinter(fieldMap map[string][]string) ([]string, error) {
	containerName := maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)

	var messageSb strings.Builder

	// Build the initial part of the message
	if containerName != "" {
		fmt.Fprintf(&messageSb, "Container '%s' image", containerName)
	} else {
		messageSb.WriteString("Image")
	}

	messageSb.WriteString(" signature is not verified by the specified signature integration(s)")

	// Check for successful verifiers and add them to the message if present
	ids, ok := fieldMap[augmentedobjs.ImageSignatureVerifiedCustomTag]
	// When no verifiers matched, there is a single item with value "<empty>". Filter it out.
	if ok && len(ids) > 0 && ids[0] != "<empty>" {
		successfulVerifiersRepr := StringSliceToSortedSentence(ids)
		fmt.Fprintf(&messageSb, " (it is verified by other integration(s): %s)", successfulVerifiersRepr)
	}

	messageSb.WriteString(".")

	return []string{messageSb.String()}, nil
}

const (
	seccompProfileTypeTemplate = `Container{{if .ContainerName}} '{{.ContainerName}}'{{end}} has Seccomp profile type '{{.SeccompProfileType}}'`
)

func seccompProfileTypePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName      string
		SeccompProfileType string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.SeccompProfileType, err = getSingleValueFromFieldMap(search.SeccompProfileType.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(seccompProfileTypeTemplate, r)
}

const (
	appArmorProfileTemplate = `Container{{if .ContainerName}} '{{.ContainerName}}'{{end}} has AppArmor profile type '{{.AppArmorProfile}}'`
)

func appArmorProfilePrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName   string
		AppArmorProfile string
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	var err error
	if r.AppArmorProfile, err = getSingleValueFromFieldMap(search.AppArmorProfile.String(), fieldMap); err != nil {
		return nil, err
	}
	return executeTemplate(appArmorProfileTemplate, r)
}

const (
	allowPrivilegeEscalationTemplate = `Container{{if .ContainerName}} '{{.ContainerName}}'{{end}} 
	{{- if .AllowPrivilegeEscalation}} allows{{else}} does not allow{{end}} privilege escalation`
)

func allowPrivilegeEscalationPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName            string
		AllowPrivilegeEscalation bool
	}
	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	allowPrivilegeEscalation, err := getSingleValueFromFieldMap(search.AllowPrivilegeEscalation.String(), fieldMap)
	if err != nil {
		return nil, err
	}
	if r.AllowPrivilegeEscalation, err = strconv.ParseBool(allowPrivilegeEscalation); err != nil {
		return nil, err
	}
	return executeTemplate(allowPrivilegeEscalationTemplate, r)
}
