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
)

func mapAlertToFinding(alert *storage.Alert) *securityhub.AwsSecurityFinding {
	severity := float64(alert.GetPolicy().GetSeverity())

	finding := &securityhub.AwsSecurityFinding{
		SchemaVersion: aws.String(schemaVersion),
		AwsAccountId:  aws.String(product.account),
		ProductArn:    aws.String(product.arn),
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
		Types: []*string{
			// TODO(tvoss): Determine proper mapping according to https://docs.aws.amazon.com/securityhub/latest/userguide/securityhub-findings-format.html#securityhub-findings-format-type-taxonomy
			aws.String("Software and Configuration Checks/Vulnerabilities/CVE"),
		},
		Resources: []*securityhub.Resource{
			// At the time of this writing, AWS security hub does not support the notion of a k8s cluster/deployment.
			// While it supports a resource type AwsEksCluster, it lacks support for cluster details.
			// With that, we instead create a custom resource and describe the deployment context of the alert in this
			// resource.
			{
				Id:   aws.String(alert.GetDeployment().GetClusterId()),
				Type: aws.String(resourceTypeOther),
				Details: &securityhub.ResourceDetails{
					Other: map[string]*string{
						"cluster-name":         aws.String(alert.GetDeployment().GetClusterName()),
						"deployment-name":      aws.String(alert.GetDeployment().GetName()),
						"deployment-namespace": aws.String(alert.GetDeployment().GetNamespace()),
					},
				},
			},
		},
		UserDefinedFields: map[string]*string{
			"cluster-name":         aws.String(alert.GetDeployment().GetClusterName()),
			"deployment-name":      aws.String(alert.GetDeployment().GetName()),
			"deployment-namespace": aws.String(alert.GetDeployment().GetNamespace()),
		},
	}

	for _, container := range alert.GetDeployment().GetContainers() {
		finding.Resources = append(finding.Resources, &securityhub.Resource{
			Id:   aws.String(container.GetImage().GetId()),
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

	switch alert.GetState() {
	case storage.ViolationState_ACTIVE, storage.ViolationState_SNOOZED:
		finding.Compliance = &securityhub.Compliance{
			Status: aws.String(securityhub.ComplianceStatusFailed),
		}
	case storage.ViolationState_RESOLVED:
		finding.Compliance = &securityhub.Compliance{
			Status: aws.String(securityhub.ComplianceStatusPassed),
		}
	}

	return finding
}

// TODO(tvoss): Fine-tune the description as we iterate on the mapping.
func createDescriptionForAlert(alert *storage.Alert) string {
	if len(alert.GetViolations()) == 0 {
		return fmt.Sprintf("Policy %s violated", alert.GetPolicy().GetName())
	}

	distinct := set.StringSet{}
	for _, v := range alert.GetViolations() {
		if vText := v.GetMessage(); vText != "" {
			distinct.Add(vText)
		}
	}

	return strings.Join(distinct.AsSortedSlice(func(a, b string) bool {
		return a < b
	}), " ")
}
