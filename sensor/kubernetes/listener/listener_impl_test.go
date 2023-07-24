package listener

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	configMocks "github.com/stackrox/rox/sensor/common/config/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	eventPipelineMocks "github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	apiVersion   = complianceoperator.GetGroupVersion().String()
	apiResources = []v1.APIResource{
		{
			Name:    complianceoperator.ProfileGVR.Resource,
			Kind:    complianceoperator.ProfileGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
		{
			Name:    complianceoperator.RuleGVR.Resource,
			Kind:    complianceoperator.RuleGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
		{
			Name:    complianceoperator.ScanSettingGVR.Resource,
			Kind:    complianceoperator.ScanSettingGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
		{
			Name:    complianceoperator.ScanSettingBindingGVR.Resource,
			Kind:    complianceoperator.ScanSettingBindingGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
		{
			Name:    complianceoperator.ComplianceScanGVR.Resource,
			Kind:    complianceoperator.ComplianceScanGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
		{
			Name:    complianceoperator.ComplianceCheckResultGVR.Resource,
			Kind:    complianceoperator.ComplianceCheckResultGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
		{
			Name:    complianceoperator.TailoredProfileGVR.Resource,
			Kind:    complianceoperator.TailoredProfileGVK.Kind,
			Group:   complianceoperator.GetGroupVersion().Group,
			Version: complianceoperator.GetGroupVersion().Version,
		},
	}
)

type fakeClientImpl struct {
	dynamic         dynamic.Interface
	k8s             kubernetes.Interface
	openshiftApps   appVersioned.Interface
	openshiftConfig configVersioned.Interface
	openshiftRoute  routeVersioned.Interface
}

func (f *fakeClientImpl) Kubernetes() kubernetes.Interface           { return f.k8s }
func (f *fakeClientImpl) Dynamic() dynamic.Interface                 { return f.dynamic }
func (f *fakeClientImpl) OpenshiftApps() appVersioned.Interface      { return f.openshiftApps }
func (f *fakeClientImpl) OpenshiftConfig() configVersioned.Interface { return f.openshiftConfig }
func (f *fakeClientImpl) OpenshiftRoute() routeVersioned.Interface   { return f.openshiftRoute }

func TestListener(t *testing.T) {
	suite.Run(t, new(ListenerTestSuite))
}

type ListenerTestSuite struct {
	suite.Suite

	k8sClient     *fake.Clientset
	dynamicClient *fakeDynamic.FakeDynamicClient
	listener      component.PipelineComponent
	resolver      *eventPipelineMocks.MockResolver
	configHandler *configMocks.MockHandler

	sensorCertDir string
}

func (s *ListenerTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceEnhancements.EnvVar())
		s.T().SkipNow()
	}

	// Setup certs
	cwd, err := os.Getwd()
	s.Require().NoError(err)
	s.T().Setenv(mtls.CAFileEnvName, filepath.Join(cwd, "testdata", "central-ca.pem"))

	ca, err := certgen.GenerateCA()
	s.Require().NoError(err)
	leafCert, err := ca.IssueCertForSubject(mtls.SensorSubject)
	s.Require().NoError(err)

	s.sensorCertDir = s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(s.sensorCertDir, "cert.pem"), leafCert.CertPEM, 0644))
	s.Require().NoError(os.WriteFile(filepath.Join(s.sensorCertDir, "key.pem"), leafCert.KeyPEM, 0600))
	s.T().Setenv(mtls.CertFilePathEnvName, filepath.Join(s.sensorCertDir, "cert.pem"))
	s.T().Setenv(mtls.KeyFileEnvName, filepath.Join(s.sensorCertDir, "key.pem"))
}

func (s *ListenerTestSuite) SetupTest() {
	s.k8sClient = fake.NewSimpleClientset()
	// Fake the compliance operator installation.
	apiResourceList := &v1.APIResourceList{GroupVersion: apiVersion, APIResources: apiResources}
	s.k8sClient.Fake.Resources = append(s.k8sClient.Fake.Resources, apiResourceList)

	apiMap := make(map[schema.GroupVersionResource]string)
	for _, resource := range apiResources {
		apiMap[schema.GroupVersionResource{
			Group:    resource.Group,
			Version:  resource.Version,
			Resource: resource.Name}] = resource.Kind + "List"
	}
	s.dynamicClient = fakeDynamic.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), apiMap)

	fakeClientI := &fakeClientImpl{
		k8s:     s.k8sClient,
		dynamic: s.dynamicClient,
	}

	mockCtrl := gomock.NewController(s.T())
	s.resolver = eventPipelineMocks.NewMockResolver(mockCtrl)
	s.configHandler = configMocks.NewMockHandler(mockCtrl)
	s.listener = New(fakeClientI, s.configHandler, "node", 0, nil, s.resolver, resources.InitializeStore())

	// Start and wait for sync.
	var wg sync.WaitGroup
	wg.Add(1)
	s.resolver.EXPECT().Send(&component.ResourceEvent{
		ForwardMessages: []*central.SensorEvent{
			{
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{},
				},
			},
		},
	}).Do(func(_ interface{}) {
		defer wg.Done()
	})
	s.Require().NoError(s.listener.Start())
	wg.Wait()
}

func (s *ListenerTestSuite) TearDownTest() {
	if s.listener != nil {
		s.listener.Stop(nil)
	}
}

func (s *ListenerTestSuite) TestEnableDisableCompliance() {
	var wg sync.WaitGroup
	var err error

	// Verify events are informed after enabling compliance.
	s.listener.Notify(common.SensorComponentEventComplianceEnabled)

	wg.Add(1)
	s.resolver.EXPECT().Send(gomock.Any()).Do(func(_ interface{}) {
		defer wg.Done()
	})
	obj := s.getTestComplianceScanObj("midnight")
	_, err = s.dynamicClient.Resource(complianceoperator.ComplianceScanGVR).Namespace("ns").Create(context.Background(), obj, v1.CreateOptions{})
	s.Require().NoError(err)
	wg.Wait()

	// Verify events are not informed after disabling compliance.
	s.listener.Notify(common.SensorComponentEventComplianceDisabled)
	time.Sleep(2 * time.Second)
	obj = s.getTestComplianceScanObj("midnight-2")
	_, err = s.dynamicClient.Resource(complianceoperator.ComplianceScanGVR).Namespace("ns").Create(context.Background(), obj, v1.CreateOptions{})
	s.Require().NoError(err)

	// Verify events are informed after enabling compliance.
	s.listener.Notify(common.SensorComponentEventComplianceEnabled)
	wg.Add(1)
	obj = s.getTestComplianceScanObj("midnight-3")
	s.resolver.EXPECT().Send(gomock.Any()).Do(func(_ interface{}) {
		defer wg.Done()
	})
	_, err = s.dynamicClient.Resource(complianceoperator.ComplianceScanGVR).Namespace("ns").Create(context.Background(), obj, v1.CreateOptions{})
	s.Require().NoError(err)
	wg.Wait()
}

func (s *ListenerTestSuite) getTestComplianceScanObj(name string) *unstructured.Unstructured {
	obj, err := k8sutil.RuntimeObjToUnstructured(&v1alpha1.ComplianceScan{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSettingGVK.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: "ns",
		},
	})
	s.Require().NoError(err)
	return obj
}

func (s *ListenerTestSuite) buildComplianceOperatorNamespace(namespace string) {
	ns := &coreV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := s.k8sClient.CoreV1().Namespaces().Create(context.Background(), ns, metaV1.CreateOptions{})
	s.Require().NoError(err)
}
