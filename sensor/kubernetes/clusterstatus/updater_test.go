package clusterstatus

import (
	"context"
	"strconv"
	"testing"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	configFake "github.com/openshift/client-go/config/clientset/versioned/fake"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type updaterSuite struct {
	suite.Suite
	updater common.SensorComponent
}

func TestClusterStatusUpdater(t *testing.T) {
	suite.Run(t, new(updaterSuite))
}

type fakeClientSet struct {
	k8s    kubernetes.Interface
	config configVersioned.Interface
}

func (c *fakeClientSet) Kubernetes() kubernetes.Interface {
	return c.k8s
}

func (c *fakeClientSet) Dynamic() dynamic.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftApps() appVersioned.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftConfig() configVersioned.Interface {
	return c.config
}

func (c *fakeClientSet) OpenshiftRoute() routeVersioned.Interface {
	return nil
}

func (c *fakeClientSet) OpenshiftOperator() operatorVersioned.Interface {
	return nil
}

func (s *updaterSuite) createUpdater(getProviders func(context.Context) *storage.ProviderMetadata,
	getMetadata providerMetadataFromOpenShift, configClient ...*configFake.Clientset) {
	config := configFake.NewSimpleClientset()
	if len(configClient) != 0 {
		config = configClient[0]
	}
	s.updater = NewUpdater(&fakeClientSet{
		k8s:    fake.NewSimpleClientset(),
		config: config,
	})
	s.updater.(*updaterImpl).getProviders = getProviders
	s.updater.(*updaterImpl).getProviderMetadataFromOpenShift = getMetadata
}

func (s *updaterSuite) online() {
	s.updater.Notify(common.SensorComponentEventCentralReachable)
}

func (s *updaterSuite) offline() {
	s.updater.Notify(common.SensorComponentEventOfflineMode)
}

func assertContextIsCancelled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		return errors.New("context is not cancelled")
	}
}

func (s *updaterSuite) readStatus() {
	msg, more := <-s.updater.ResponsesC()
	s.Assert().True(more, "channel should be open")
	s.Assert().False(msg.IsExpired(), "message should not be expired")
	s.Assert().NotNil(msg.GetClusterStatusUpdate().GetStatus(), "message should be ClusterStatus")
}

func (s *updaterSuite) readCancelledStatus() {
	updater, ok := s.updater.(*updaterImpl)
	s.Require().True(ok)
	select {
	case msg, more := <-s.updater.ResponsesC():
		s.Assert().True(more, "channel should be open")
		s.Assert().True(msg.IsExpired(), "message should not be expired")
		s.Assert().NotNil(msg.GetClusterStatusUpdate().GetStatus(), "message should be ClusterStatus")
	case <-time.After(10 * time.Nanosecond):
		// If context is cancelled the message might not be sent at all
		s.Assert().NoError(assertContextIsCancelled(updater.getCurrentContext()))
	}
}

func (s *updaterSuite) readDeploymentEnv() {
	msg, more := <-s.updater.ResponsesC()
	s.Assert().True(more, "channel should be open")
	s.Assert().False(msg.IsExpired(), "message should not be expired")
	s.Assert().NotNil(msg.GetClusterStatusUpdate().GetDeploymentEnvUpdate(), "message should be DeploymentEnvUpdate")
}

func (s *updaterSuite) readCancelledDeploymentEnv() {
	updater, ok := s.updater.(*updaterImpl)
	s.Require().True(ok)
	select {
	case msg, more := <-s.updater.ResponsesC():
		s.Assert().True(more, "channel should be open")
		s.Assert().True(msg.IsExpired(), "message should not be expired")
		s.Assert().NotNil(msg.GetClusterStatusUpdate().GetDeploymentEnvUpdate(), "message should be DeploymentEnvUpdate")
	case <-time.After(10 * time.Nanosecond):
		// If context is cancelled the message might not be sent at all
		s.Assert().NoError(assertContextIsCancelled(updater.getCurrentContext()))
	}
}

func mockGetMetadata(_ context.Context) *storage.ProviderMetadata {
	return &storage.ProviderMetadata{}
}

func mockProviderMetadata(_ context.Context, _ configVersioned.Interface) (*storage.ProviderMetadata, error) {
	return nil, nil
}

func (s *updaterSuite) Test_OfflineMode() {
	cases := map[string][]func(){
		"Online, offline, read":                           {s.online, s.offline, s.readCancelledStatus},
		"Online, read, offline, read":                     {s.online, s.readStatus, s.offline, s.readCancelledDeploymentEnv},
		"Online, read, read, offline, online, read, read": {s.online, s.readStatus, s.readDeploymentEnv, s.offline, s.online, s.readStatus, s.readDeploymentEnv},
	}
	for tName, tc := range cases {
		s.Run(tName, func() {
			s.createUpdater(mockGetMetadata, mockProviderMetadata)
			for _, fn := range tc {
				fn()
			}
		})
	}
}

func (s *updaterSuite) Test_GetCloudProviderMetadata() {
	testProviderMetadata := &storage.ProviderMetadata{
		Region: "us-east1",
		Zone:   "us-east1-a",
		Provider: &storage.ProviderMetadata_Google{Google: &storage.GoogleProviderMetadata{
			Project:     "sample-thing",
			ClusterName: "sample-cluster",
		}},
		Verified: true,
		Cluster: &storage.ClusterMetadata{
			Type: storage.ClusterMetadata_GKE,
			Name: "sample-cluster",
			Id:   "1",
		},
	}

	nilGetProviders := func(_ context.Context) *storage.ProviderMetadata { return nil }

	infraTypeMeta := metav1.TypeMeta{Kind: "Infrastructure", APIVersion: "config.openshift.io/v1"}
	infraObjectMeta := metav1.ObjectMeta{
		Name: "cluster",
	}
	cvTypeMeta := metav1.TypeMeta{Kind: "ClusterVersion", APIVersion: "config.openshift.io/v1"}
	cvObjectMeta := metav1.ObjectMeta{Name: "version"}

	cases := map[string]struct {
		infra        *configv1.Infrastructure
		cv           *configv1.ClusterVersion
		metadata     *storage.ProviderMetadata
		getProviders func(ctx context.Context) *storage.ProviderMetadata
		openshift    bool
	}{
		"return of provider should not call any k8s API": {
			getProviders: func(ctx context.Context) *storage.ProviderMetadata {
				return testProviderMetadata
			},
			metadata: testProviderMetadata,
		},
		"no provider returned from get providers and not running on OpenShift should return nil": {
			getProviders: nilGetProviders,
		},
		"on openshift running on AWS should return AWS provider metadata": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.AWSPlatformType,
						AWS: &configv1.AWSPlatformStatus{
							Region: "us-east1",
						}},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region:   "us-east1",
				Provider: &storage.ProviderMetadata_Aws{Aws: &storage.AWSProviderMetadata{}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_OCP,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running on GCP should return GCP provider metadata": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.GCPPlatformType,
						GCP: &configv1.GCPPlatformStatus{
							ProjectID: "project-1",
							Region:    "us-east1",
						}},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region: "us-east1",
				Provider: &storage.ProviderMetadata_Google{Google: &storage.GoogleProviderMetadata{
					Project: "project-1",
				}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_OCP,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running on Azure should return Azure provider metadata": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type:  configv1.AzurePlatformType,
						Azure: &configv1.AzurePlatformStatus{},
					},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region:   "",
				Provider: &storage.ProviderMetadata_Azure{Azure: &storage.AzureProviderMetadata{}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_OCP,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running on a provider not supported should return basic information": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type:         configv1.AlibabaCloudPlatformType,
						AlibabaCloud: &configv1.AlibabaCloudPlatformStatus{},
					},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_OCP,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running OSD on AWS should return AWS provider metadata and OSD cluster type": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.AWSPlatformType,
						AWS: &configv1.AWSPlatformStatus{
							Region: "us-east1",
							ResourceTags: []configv1.AWSResourceTag{
								{
									Key:   redHatClusterTypeTagKey,
									Value: "osd",
								},
							},
						}},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region:   "us-east1",
				Provider: &storage.ProviderMetadata_Aws{Aws: &storage.AWSProviderMetadata{}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_OSD,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running OSD on GCP should return GCP provider metadata and OSD cluster type": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.GCPPlatformType,
						GCP: &configv1.GCPPlatformStatus{
							ProjectID: "project-1",
							Region:    "us-east1",
							ResourceTags: []configv1.GCPResourceTag{
								{
									Key:   redHatClusterTypeTagKey,
									Value: "osd",
								},
							},
						}},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region: "us-east1",
				Provider: &storage.ProviderMetadata_Google{Google: &storage.GoogleProviderMetadata{
					Project: "project-1",
				}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_OSD,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running ROSA on AWS should return AWS provider metadata and ROSA cluster type": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.AWSPlatformType,
						AWS: &configv1.AWSPlatformStatus{
							Region: "us-east1",
							ResourceTags: []configv1.AWSResourceTag{
								{
									Key:   redHatClusterTypeTagKey,
									Value: "rosa",
								},
							},
						}},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region:   "us-east1",
				Provider: &storage.ProviderMetadata_Aws{Aws: &storage.AWSProviderMetadata{}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_ROSA,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
		"on openshift running ARO on Azure should return Azure provider metadata and ARO cluster type": {
			getProviders: nilGetProviders,
			openshift:    true,
			infra: &configv1.Infrastructure{
				TypeMeta:   infraTypeMeta,
				ObjectMeta: infraObjectMeta,
				Status: configv1.InfrastructureStatus{
					PlatformStatus: &configv1.PlatformStatus{
						Type: configv1.AzurePlatformType,
						Azure: &configv1.AzurePlatformStatus{
							ResourceTags: []configv1.AzureResourceTag{
								{
									Key:   redHatClusterTypeTagKey,
									Value: "aro",
								},
							},
						},
					},
					InfrastructureName: "cluster-1",
				},
			},
			cv: &configv1.ClusterVersion{
				TypeMeta:   cvTypeMeta,
				ObjectMeta: cvObjectMeta,
				Spec:       configv1.ClusterVersionSpec{ClusterID: "44a6254c-8bc4-4724-abfe-c510747742b8"},
			},
			metadata: &storage.ProviderMetadata{
				Region:   "",
				Provider: &storage.ProviderMetadata_Azure{Azure: &storage.AzureProviderMetadata{}},
				Verified: true,
				Cluster: &storage.ClusterMetadata{
					Type: storage.ClusterMetadata_ARO,
					Name: "cluster-1",
					Id:   "44a6254c-8bc4-4724-abfe-c510747742b8",
				},
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			s.T().Setenv(env.OpenshiftAPI.EnvVar(), strconv.FormatBool(tc.openshift))
			config := configFake.NewSimpleClientset()
			if tc.infra != nil {
				config = configFake.NewSimpleClientset(tc.infra, tc.cv)
			}
			s.createUpdater(tc.getProviders, getProviderMetadataFromOpenShiftConfig, config)
			u := s.updater.(*updaterImpl)
			providerMetadata := u.getCloudProviderMetadata(context.Background())
			protoassert.Equal(s.T(), tc.metadata, providerMetadata)
		})
	}
}
