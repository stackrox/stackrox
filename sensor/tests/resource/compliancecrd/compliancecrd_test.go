package compliancecrd

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/tests/helper"
	"github.com/stretchr/testify/suite"
	apiextensionv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
)

var (
	schema = apiextensionv1.CustomResourceValidation{
		OpenAPIV3Schema: &apiextensionv1.JSONSchemaProps{
			Description: "ComplianceCheckResult represent a result of a single compliance",
			Properties: map[string]apiextensionv1.JSONSchemaProps{
				"apiVersion": {
					Description: "'APIVersion defines the versioned schema of this representation\n              of an object. Servers should convert recognized schemas to the latest\n              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'",
					Type:        "string",
				},
				"description": {
					Description: "A human-readable check description, what and why it does",
					Type:        "string",
				},
				"id": {
					Description: "A unique identifier of a check",
					Type:        "string",
				},
				"instructions": {
					Description: "How to evaluate if the rule status manually. If no automatic\n              test is present, the rule status will be MANUAL and the administrator\n              should follow these instructions.",
					Type:        "string",
				},
				"kind": {
					Description: "'Kind is a string value representing the REST resource this\n              object represents. Servers may infer this from the endpoint the client\n              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'",
					Type:        "string",
				},
				"metadata": {
					Type: "object",
				},
				"severity": {
					Description: "The severity of a check status",
					Type:        "string",
				},
				"status": {
					Description: "The result of a check",
					Type:        "string",
				},
				"valuesUsed": {
					Description: "It stores a list of values used by the check",
					Type:        "array",
					Items: &apiextensionv1.JSONSchemaPropsOrArray{
						Schema: &apiextensionv1.JSONSchemaProps{
							Type: "string",
						},
					},
				},
				"warnings": {
					Description: "Any warnings that the user should be aware about.",
					Type:        "array",
					Items: &apiextensionv1.JSONSchemaPropsOrArray{
						Schema: &apiextensionv1.JSONSchemaProps{
							Type: "string",
						},
					},
					Nullable: true,
				},
			},
			Required: []string{"id", "severity", "status"},
			Type:     "object",
		},
	}
	complianceCRD = apiextensionv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "compliancecheckresults.compliance.openshift.io",
		},
		Spec: apiextensionv1.CustomResourceDefinitionSpec{
			Group: "compliance.openshift.io",
			Scope: apiextensionv1.NamespaceScoped,
			Names: apiextensionv1.CustomResourceDefinitionNames{
				Plural:     "compliancecheckresults",
				Kind:       "ComplianceCheckResult",
				Singular:   "compliancecheckresult",
				ShortNames: []string{"ccr", "checkresults", "checkresult"},
				ListKind:   "ComplianceCheckResultList",
			},
			Versions: []apiextensionv1.CustomResourceDefinitionVersion{
				{
					AdditionalPrinterColumns: []apiextensionv1.CustomResourceColumnDefinition{
						{
							JSONPath: ".status",
							Name:     "Status",
							Type:     "string",
						},
						{
							JSONPath: ".severity",
							Name:     "Severity",
							Type:     "string",
						},
					},
					Name:    "v1alpha1",
					Served:  true,
					Storage: true,
					Schema:  &schema,
				},
			},
		},
	}
)

type ComplianceCRDSuite struct {
	suite.Suite
	testContext *helper.TestContext
}

func Test_ComplianceCRD(t *testing.T) {
	suite.Run(t, new(ComplianceCRDSuite))
}

func (s *ComplianceCRDSuite) SetupSuite() {
	testContext, err := helper.NewContextWithConfig(s.T(), helper.DefaultConfig())
	s.Require().NoError(err)
	s.testContext = testContext
}

func (s *ComplianceCRDSuite) TearDownTest() {
	s.testContext.GetFakeCentral().ClearReceivedBuffer()
}

func (s *ComplianceCRDSuite) Test_StopOnCRDsDetected() {
	s.T().Setenv(env.ComplianceCRDsWatchTimer.EnvVar(), "10ms")
	s.testContext.RunTest(s.T(),
		helper.WithTestCase(func(t *testing.T, tc *helper.TestContext, resource map[string]k8s.Object) {
			// Wait for sync event
			tc.WaitForSyncEvent(t, 10*time.Second)
			// Create Compliance CRD
			ctx := context.Background()
			cli, err := apiextension.NewForConfig(tc.GetK8sConfig())
			s.Require().NoError(err)
			res, err := cli.CustomResourceDefinitions().Create(ctx, &complianceCRD, metav1.CreateOptions{})
			s.Require().NoError(err)
			defer func() {
				err = cli.CustomResourceDefinitions().Delete(ctx, res.GetName(), metav1.DeleteOptions{})
				s.Require().NoError(err)
			}()
			// Wait for sensor to stop
			tc.WaitForStop(t, 10*time.Second)
		}))
}
