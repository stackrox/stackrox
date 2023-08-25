package awssh

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/set"
)

const (
	iso8601UTC = "2006-01-02T15:04:05Z"
	// criticalSeverity is used to normalize the severity of an alert.
	criticalSeverity      = float64(storage.Severity_CRITICAL_SEVERITY)
	schemaVersion         = "2018-10-08"
	resourceTypeContainer = "Container"
	resourceTypeOther     = "Other"

	maxResources = 32
)

var (
	categoryMap = map[string]string{
		"vulnerability management": "Software and Configuration Checks/Vulnerabilities/CVE",
		"anomalous activity":       "Unusual Behaviors/Container",
		"cryptocurrency mining":    "Unusual Behaviors/Container",
		"devops best practices":    "TTPs",
		"kubernetes":               "TTPs",
		"network tools":            "Unusual Behaviors/Network Flows",
		"package management":       "TTPs",
		"privileges":               "TTPs/Privilege Escalation",
		"security best practices":  "TTPs",
		"system modification":      "Effects",
		"kubernetes events":        "Unusual Behaviors",
		"docker cis":               "Software and Configuration Checks/Industry and Regulatory Standards",
	}

	defaultFindingType = "TTPs"
)

func categoriesToFindings(categories []string) []*string {
	if len(categories) == 0 {
		return []*string{
			aws.String(defaultFindingType),
		}
	}
	findingSet := set.NewStringSet()
	for _, category := range categories {
		if awsFindingType, ok := categoryMap[strings.ToLower(category)]; ok {
			findingSet.Add(awsFindingType)
			continue
		}
		findingSet.Add(defaultFindingType)
	}
	findings := make([]*string, 0, len(findingSet))
	for finding := range findingSet {
		findings = append(findings, aws.String(finding))
	}
	return findings
}

func mapAlertToFinding(account string, arn string, alertURL string, alert *storage.Alert) *securityhub.AwsSecurityFinding {
	severity := float64(alert.GetPolicy().GetSeverity())

	finding := &securityhub.AwsSecurityFinding{
		SchemaVersion: aws.String(schemaVersion),
		AwsAccountId:  aws.String(account),
		ProductArn:    aws.String(arn),
		// See https://docs.aws.amazon.com/securityhub/latest/userguide/securityhub-custom-providers.html
		ProductFields: map[string]*string{
			"ProviderName":    aws.String(product.name),
			"ProviderVersion": aws.String(product.version),
		},
		GeneratorId: aws.String(alert.GetPolicy().GetId()),
		Id:          aws.String(alert.GetId()),
		Title:       aws.String(fmt.Sprintf("Policy %s violated", alert.GetPolicy().GetName())),
		Description: aws.String(createDescriptionForAlert(alert)),
		CreatedAt:   aws.String(protoconv.ConvertTimestampToTimeOrNow(alert.GetFirstOccurred()).UTC().Format(iso8601UTC)),
		UpdatedAt:   aws.String(protoconv.ConvertTimestampToTimeOrNow(alert.GetTime()).UTC().Format(iso8601UTC)),
		Confidence:  aws.Int64(100),
		Severity: &securityhub.Severity{
			Normalized: aws.Int64(int64(100 * severity / criticalSeverity)),
			Product:    aws.Float64(severity),
		},
		Types:     categoriesToFindings(alert.GetPolicy().GetCategories()),
		SourceUrl: aws.String(alertURL),
	}

	resources := getEntitySection(alert)

	for i, violation := range alert.GetViolations() {
		resources = append(resources, &securityhub.Resource{
			Id:   aws.String("violation: " + violation.GetMessage()),
			Type: aws.String(resourceTypeOther),
		})
		// If we are going to add eclipse the maxResource limit, the use the last entry to
		// reference the StackRox UI and break
		if len(resources) == maxResources-1 && i != len(alert.GetViolations())-1 {
			resources = append(resources, &securityhub.Resource{
				Id:   aws.String(fmt.Sprintf("Note: More than %d violations found. Please consult the StackRox product to see more.", maxResources)),
				Type: aws.String(resourceTypeOther),
			})
			break
		}
	}
	finding.Resources = resources

	switch alert.GetState() {
	case storage.ViolationState_ACTIVE, storage.ViolationState_SNOOZED:
		finding.Compliance = &securityhub.Compliance{
			Status: aws.String(securityhub.ComplianceStatusFailed),
		}
		finding.RecordState = aws.String("ACTIVE")
	case storage.ViolationState_RESOLVED:
		finding.Compliance = &securityhub.Compliance{
			Status: aws.String(securityhub.ComplianceStatusPassed),
		}
		finding.SetRecordState("ARCHIVED")
	}

	return finding
}

func getEntitySection(alert *storage.Alert) []*securityhub.Resource {
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return getDeploymentSection(entity.Deployment)
	case *storage.Alert_Resource_:
		return getResourceSection(entity.Resource)
	}
	return nil
}

func getResourceSection(resource *storage.Alert_Resource) []*securityhub.Resource {
	resources := []*securityhub.Resource{
		// At the time of this writing, AWS security hub does not support the notion of a k8s cluster/deployment.
		// While it supports a resource type AwsEksCluster, it lacks support for cluster details.
		// With that, we instead create a custom resource and describe the k8s resource context of the alert in this
		// resource.
		{
			Id:   aws.String(fmt.Sprintf("resource: %s", resource.GetName())),
			Type: aws.String(resourceTypeOther),
			Details: &securityhub.ResourceDetails{
				Other: map[string]*string{
					"cluster-name":       aws.String(resource.GetClusterName()),
					"resource-name":      aws.String(resource.GetName()),
					"resource-namespace": aws.String(resource.GetNamespace()),
					"resource-type":      aws.String(resource.GetResourceType().String()),
				},
			},
		},
	}
	return resources
}

func getDeploymentSection(deployment *storage.Alert_Deployment) []*securityhub.Resource {
	resources := []*securityhub.Resource{
		// At the time of this writing, AWS security hub does not support the notion of a k8s cluster/deployment.
		// While it supports a resource type AwsEksCluster, it lacks support for cluster details.
		// With that, we instead create a custom resource and describe the deployment context of the alert in this
		// resource.
		{
			Id:   aws.String(fmt.Sprintf("deployment: %s", deployment.GetName())),
			Type: aws.String(resourceTypeOther),
			Details: &securityhub.ResourceDetails{
				Other: map[string]*string{
					"cluster-name":         aws.String(deployment.GetClusterName()),
					"deployment-name":      aws.String(deployment.GetName()),
					"deployment-namespace": aws.String(deployment.GetNamespace()),
				},
			},
		},
	}

	for _, container := range deployment.GetContainers() {
		if container.GetImage().GetId() == "" {
			continue
		}
		resources = append(resources, &securityhub.Resource{
			Id:   aws.String(fmt.Sprintf("container: %s.%s@%s: %s", deployment.GetName(), deployment.GetNamespace(), deployment.GetClusterName(), container.GetName())),
			Type: aws.String(resourceTypeContainer),
			Details: &securityhub.ResourceDetails{
				Container: &securityhub.ContainerDetails{
					Name:      aws.String(container.GetName()),
					ImageId:   aws.String(container.GetImage().GetId()),
					ImageName: aws.String(container.GetImage().GetName().GetFullName()),
				},
			},
		})
	}
	return resources
}

// TODO(tvoss): Fine-tune the description as we iterate on the mapping.
func createDescriptionForAlert(alert *storage.Alert) string {
	s := alert.GetPolicy().GetDescription()
	if len(s) > 1024 {
		s = s[:1024]
	}
	if s == "" {
		return "<policy has no description>"
	}
	return s
}
