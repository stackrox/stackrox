//go:build sql_integration

package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/utils/pointer"
)

type label struct {
	key   string
	value string
}

var (
	collectionNamespaces = []string{
		"collections-test-1",
		"collections-test-2",
	}
	testDeployments = []*appsv1.Deployment{
		getTestDeployment("deployment1", []label{
			{"key1", "value1"},
		}...),
		getTestDeployment("deployment2", []label{
			{"key1", "value1"},
			{"key2", "value2"},
		}...),
		getTestDeployment("deployment3", []label{
			{"key2", "value2"},
		}...),
		getTestDeployment("deployment4", []label{
			{"key1", "value11"},
		}...),
	}
)

func TestCollectionE2E(t *testing.T) {
	suite.Run(t, new(CollectionE2ETestSuite))
}

type CollectionE2ETestSuite struct {
	suite.Suite

	ctx           context.Context
	nsClient      k8scorev1.NamespaceInterface
	service       v1.CollectionServiceClient
	depService    v1.DeploymentServiceClient
	collectionIDs []string
}

func (s *CollectionE2ETestSuite) SetupSuite() {

	var err error

	s.ctx = context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(s.T())
	s.service = v1.NewCollectionServiceClient(conn)
	s.depService = v1.NewDeploymentServiceClient(conn)
	k8sInterface := createK8sClient(s.T())

	// create testing namespaces
	s.nsClient = k8sInterface.CoreV1().Namespaces()
	for _, ns := range collectionNamespaces {
		_, err = s.nsClient.Create(s.ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		}, metav1.CreateOptions{})
		s.NoError(err)

		// load deployments into created namespace
		depClient := k8sInterface.AppsV1().Deployments(ns)
		for _, dep := range testDeployments {
			_, err = depClient.Create(s.ctx, dep, metav1.CreateOptions{})
			s.NoError(err)
		}
	}

	// upsert some collections to use as embedded
	createCollectionRequests := []*v1.CreateCollectionRequest{
		{
			Name: "dep1",
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.DeploymentName.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "deployment1",
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "dep2",
			ResourceSelectors: []*storage.ResourceSelector{
				{
					Rules: []*storage.SelectorRule{
						{
							FieldName: search.DeploymentName.String(),
							Operator:  storage.BooleanOperator_OR,
							Values: []*storage.RuleValue{
								{
									Value: "deployment2",
								},
							},
						},
					},
				},
			},
		},
	}
	for _, req := range createCollectionRequests {
		resp, err := s.service.CreateCollection(s.ctx, req)
		s.NoError(err)
		s.collectionIDs = append(s.collectionIDs, resp.GetCollection().GetId())
	}

	// wait for deployments to propagate
	qb := search.NewQueryBuilder().AddRegexes(search.Namespace, "collections-test-.")
	waitForDeploymentCount(s.T(), qb.Query(), len(collectionNamespaces)*len(testDeployments))
}

func (s *CollectionE2ETestSuite) TearDownSuite() {
	// clean up namespaces
	for _, ns := range collectionNamespaces {
		err := s.nsClient.Delete(s.ctx, ns, metav1.DeleteOptions{})
		if err != nil {
			log.Errorf("failed deleting %q testing namespace %q", ns, err)
		}
	}

	// clean up collections
	for _, id := range s.collectionIDs {
		_, err := s.service.DeleteCollection(s.ctx, &v1.ResourceByID{Id: id})
		if err != nil {
			log.Errorf("failed deleting %q testing collection %q", id, err)
		}
	}
}

func (s *CollectionE2ETestSuite) TestDeploymentMatching() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	// get deployments to test against
	deploymentQuery := search.NewQueryBuilder().AddExactMatches(search.Namespace, collectionNamespaces...).Query()
	deploymentQueryResponse, err := s.depService.ListDeployments(ctx, &v1.RawQuery{Query: deploymentQuery})
	s.NoError(err)
	deploymentList := deploymentQueryResponse.GetDeployments()

	// create filter query
	filterQuery, err := search.NewQueryBuilder().AddExactMatches(search.Namespace, collectionNamespaces[0]).RawQuery()
	s.NoError(err)

	// test cases
	for _, tc := range []struct {
		name                    string
		request                 *v1.DryRunCollectionRequest
		expectedListDeployments []*storage.ListDeployment
	}{
		{
			"deployment name",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "deployment1",
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1"
			}),
		},
		{
			"deployment names or",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "deployment1",
									},
									{
										Value: "deployment2",
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1" || deployment.GetName() == "deployment2"
			}),
		},
		{
			"deployment names and",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "deployment1",
									},
								},
							},
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "deployment2",
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			[]*storage.ListDeployment{},
		},
		{
			"deployment label",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentLabel.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value:     "key1=value1",
										MatchType: storage.MatchType_EXACT,
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1" || deployment.GetName() == "deployment2"
			}),
		},
		{
			"deployment labels or",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentLabel.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "key1=value1",
									},
									{
										Value: "key2=value2",
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1" || deployment.GetName() == "deployment2" || deployment.GetName() == "deployment3"
			}),
		},
		{
			"deployment labels and",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentLabel.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "key1=value1",
									},
								},
							},
							{
								FieldName: search.DeploymentLabel.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "key2=value2",
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment2"
			}),
		},
		{
			"namespace with embedded",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.Namespace.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: collectionNamespaces[0],
									},
								},
							},
						},
					},
				},
				EmbeddedCollectionIds: []string{s.collectionIDs[0]},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1" || deployment.GetNamespace() == collectionNamespaces[0]
			}),
		},
		{
			"namespace with deployment name",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.Namespace.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: collectionNamespaces[0],
									},
								},
							},
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "deployment2",
									},
								},
							},
						},
					},
				},
				EmbeddedCollectionIds: []string{s.collectionIDs[0]},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1" || (deployment.GetNamespace() == collectionNamespaces[0] && deployment.GetName() == "deployment2")
			}),
		},
		{
			"regex matching",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.Namespace.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: collectionNamespaces[0],
									},
									{
										Value: collectionNamespaces[1],
									},
								},
							},
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value:     ".*2",
										MatchType: storage.MatchType_REGEX,
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment2"
			}),
		},
		{
			"filter query",
			&v1.DryRunCollectionRequest{
				Name: "test collection",
				ResourceSelectors: []*storage.ResourceSelector{
					{
						Rules: []*storage.SelectorRule{
							{
								FieldName: search.DeploymentName.String(),
								Operator:  storage.BooleanOperator_OR,
								Values: []*storage.RuleValue{
									{
										Value: "deployment2",
									},
								},
							},
						},
					},
				},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
					FilterQuery: &v1.RawQuery{Query: filterQuery},
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment2" && deployment.GetNamespace() == collectionNamespaces[0]
			}),
		},
		{
			"embedded collection",
			&v1.DryRunCollectionRequest{
				Name:                  "test collection",
				EmbeddedCollectionIds: []string{s.collectionIDs[1]},
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment2"
			}),
		},
		{
			"embedded collections",
			&v1.DryRunCollectionRequest{
				Name:                  "test collection",
				EmbeddedCollectionIds: s.collectionIDs,
				Options: &v1.CollectionDeploymentMatchOptions{
					WithMatches: true,
				},
			},
			filter(deploymentList, func(deployment *storage.ListDeployment) bool {
				return deployment.GetName() == "deployment1" || deployment.GetName() == "deployment2"
			}),
		},
	} {
		s.T().Run(tc.name, func(t *testing.T) {
			resp, err := s.service.DryRunCollection(ctx, tc.request)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tc.expectedListDeployments, resp.GetDeployments())
		})
	}
}

func getTestDeployment(name string, labels ...label) *appsv1.Deployment {
	ret := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "quay.io/rhacs-eng/qa:nginx-1-14-alpine",
						},
					},
				},
			},
		},
	}

	// set labels
	for _, l := range labels {
		ret.Spec.Selector.MatchLabels[l.key] = l.value
		ret.Spec.Template.ObjectMeta.Labels[l.key] = l.value
		ret.ObjectMeta.Labels[l.key] = l.value
	}
	return ret
}

func filter[T any](list []T, test func(T) bool) (ret []T) {
	for _, obj := range list {
		if test(obj) {
			ret = append(ret, obj)
		}
	}
	return
}
