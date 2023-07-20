package complianceoperator

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/suite"
	appsV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	// Max time to receive info status. You may want to increase it if you plan to step through the code with debugger.
	responseTimeout = 5 * time.Second
	defaultNS       = "openshift-compliance"
	customNS        = "la-la-land"
)

func TestUpdater(t *testing.T) {
	suite.Run(t, new(UpdaterTestSuite))
}

type UpdaterTestSuite struct {
	suite.Suite

	client *fake.Clientset
}

type expectedInfo struct {
	version        string
	namespace      string
	desired, ready int32
	error          string
}

func (s *UpdaterTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceEnhancements.EnvVar())
		s.T().SkipNow()
	}
}

func (s *UpdaterTestSuite) SetupTest() {
	s.client = fake.NewSimpleClientset()
	_, err := s.client.CoreV1().Namespaces().Create(context.Background(), buildComplianceOperatorNamespace(defaultNS), metaV1.CreateOptions{})
	s.Require().NoError(err)

	_, err = s.client.CoreV1().Namespaces().Create(context.Background(), buildComplianceOperatorNamespace(customNS), metaV1.CreateOptions{})
	s.Require().NoError(err)
}

func (s *UpdaterTestSuite) TestDefaultNamespace() {
	ds := buildComplianceOperator(defaultNS)

	s.createCO(ds)

	actual := s.getInfo(1, 1*time.Millisecond)
	// Compliance operator found, CRDs not found.
	s.assertEqual(expectedInfo{
		"v1.0.0", defaultNS, 1, 1,
		"the server could not find the requested resource, GroupVersion \"compliance.openshift.io/v1alpha1\" not found",
	}, actual)
}

func (s *UpdaterTestSuite) TestMultipleTries() {
	ds := buildComplianceOperator(defaultNS)
	s.createCO(ds)

	actual := s.getInfo(3, 1*time.Millisecond)
	// Compliance operator found, CRDs not found.
	s.assertEqual(expectedInfo{
		"v1.0.0", defaultNS, 1, 1,
		"the server could not find the requested resource, GroupVersion \"compliance.openshift.io/v1alpha1\" not found",
	}, actual)
}

func (s *UpdaterTestSuite) TestNotFound() {
	actual := s.getInfo(1, 1*time.Millisecond)
	s.assertEqual(expectedInfo{error: "deployment compliance-operator not found in any namespace"}, actual)
}

func (s *UpdaterTestSuite) TestDelayedTicker() {
	ds := buildComplianceOperator(defaultNS)

	s.createCO(ds)

	actual := s.getInfo(1, 1*time.Minute)
	// Compliance operator found, CRDs not found.
	s.assertEqual(expectedInfo{
		"v1.0.0", defaultNS, 1, 1,
		"the server could not find the requested resource, GroupVersion \"compliance.openshift.io/v1alpha1\" not found",
	}, actual)
}

func (s *UpdaterTestSuite) getInfo(times int, updateInterval time.Duration) *central.ComplianceOperatorInfo {
	timer := time.NewTimer(responseTimeout)
	updater := NewInfoUpdater(s.client, updateInterval)

	err := updater.Start()
	s.Require().NoError(err)
	defer updater.Stop(nil)

	var info *central.ComplianceOperatorInfo

	for i := 0; i < times; i++ {
		select {
		case response := <-updater.ResponsesC():
			info = response.Msg.(*central.MsgFromSensor_ComplianceOperatorInfo).ComplianceOperatorInfo
		case <-timer.C:
			s.Fail("Timed out while waiting for compliance operator info")
		}
	}

	return info
}

func buildComplianceOperatorNamespace(namespace string) *coreV1.Namespace {
	return &coreV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: namespace,
		},
	}
}

func buildComplianceOperator(namespace string) *appsV1.Deployment {
	return &appsV1.Deployment{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "compliance-operator",
			Namespace: namespace,
			Labels: map[string]string{
				"olm.owner": "compliance-operator.v1.0.0",
			},
		},
		Spec: appsV1.DeploymentSpec{
			Template: coreV1.PodTemplateSpec{
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:  "compliance-operator",
							Image: "registry.redhat.io/compliance/openshift-compliance-rhel8-operator@sha256:6cd0ea9ff7102213b41ae0a4d181d75b5d76febf1287164ddbb15133560fe1a1",
						},
					},
				},
			},
		},
		Status: appsV1.DeploymentStatus{
			Replicas:      1,
			ReadyReplicas: 1,
		},
	}
}

func (s *UpdaterTestSuite) createCO(ds *appsV1.Deployment) {
	_, err := s.client.AppsV1().Deployments(ds.ObjectMeta.Namespace).Create(context.Background(), ds, metaV1.CreateOptions{})
	s.Require().NoError(err)

	ds, err = s.client.AppsV1().Deployments(ds.ObjectMeta.Namespace).Get(context.Background(), complianceoperator.Name, metaV1.GetOptions{})
	s.Require().NoError(err)
	s.Require().Equal(ds.Name, complianceoperator.Name)
}

func (s *UpdaterTestSuite) assertEqual(expected expectedInfo, actual *central.ComplianceOperatorInfo) {
	expectedVal := &central.ComplianceOperatorInfo{
		Version:     expected.version,
		Namespace:   expected.namespace,
		StatusError: expected.error,
	}

	if expected.desired > 0 {
		expectedVal.TotalDesiredPodsOpt = &central.ComplianceOperatorInfo_TotalDesiredPods{
			TotalDesiredPods: expected.desired,
		}
	}
	if expected.ready > 0 {
		expectedVal.TotalReadyPodsOpt = &central.ComplianceOperatorInfo_TotalReadyPods{
			TotalReadyPods: expected.ready,
		}
	}
	s.EqualValues(expectedVal, actual)
}
