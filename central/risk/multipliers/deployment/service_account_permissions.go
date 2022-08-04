package deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"

	roleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/rbac/utils"
	serviceAccountStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

var (
	rbacConfigurationHeading = "RBAC Configuration"

	namespaceReadWeight  = 1.0
	namespaceWriteWeight = 2.0
	clusterReadWeight    = 3.0
	clusterWriteWeight   = 4.0

	maxResources       = 46.0
	maxPermissionScore = 1518.0

	riskFactorPrefix     = "Service account %q has been granted"
	clusterAdminSuffix   = "cluster admin privileges in the cluster"
	clusterScopeSuffix   = "the following permissions on resources in the cluster:"
	namespaceScopeSuffix = "the following permissions on resources in namespace %q:"
)

// saPermissionsMultiplier is a scorer for the permissions granted to the service account of the deployment
type saPermissionsMultiplier struct {
	roleStore           roleStore.DataStore
	bindingStore        bindingStore.DataStore
	serviceAccountStore serviceAccountStore.DataStore
}

// NewSAPermissionsMultiplier provides a multiplier that scores the data based on the the number and type of permissions granted to the service account
func NewSAPermissionsMultiplier(roleStore roleStore.DataStore, bindingStore bindingStore.DataStore, saStore serviceAccountStore.DataStore) Multiplier {
	return &saPermissionsMultiplier{
		roleStore:           roleStore,
		bindingStore:        bindingStore,
		serviceAccountStore: saStore,
	}
}

// Score takes a deployment and evaluates its risk based on the permissions granted to the deployment's service account
func (c *saPermissionsMultiplier) Score(ctx context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	// TODO(ROX-9637)
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	var factors []*storage.Risk_Result_Factor
	overallScore := float32(0)

	autoMountFactors, autoMounted := c.tokenAutomounted(ctx, deployment)
	if !autoMounted {
		return nil
	}

	factors = append(factors, autoMountFactors...)
	overallScore = 1.0

	subject := k8srbac.GetSubjectForDeployment(deployment)
	clusterScore, verbs, isAdmin := c.getClusterPermissionsScore(ctx, deployment, subject)

	if isAdmin {
		factors = append(factors, &storage.Risk_Result_Factor{
			Message: fmt.Sprintf(strings.Join([]string{riskFactorPrefix, clusterAdminSuffix}, " "), subject.GetName()),
		})
		overallScore += float32(clusterScore) / float32(maxPermissionScore)

	} else {
		if clusterScore > 0.0 {
			factors = append(factors, &storage.Risk_Result_Factor{
				Message: fmt.Sprintf(strings.Join([]string{riskFactorPrefix, clusterScopeSuffix, verbs}, " "), subject.GetName()),
			})
		}

		namespaceScore, verbs := c.getNamespacePermissionsScore(ctx, deployment, subject)
		if namespaceScore > 0.0 {
			factors = append(factors, &storage.Risk_Result_Factor{
				Message: fmt.Sprintf(strings.Join([]string{riskFactorPrefix, namespaceScopeSuffix, verbs}, " "),
					subject.GetName(), deployment.GetNamespace()),
			})
		}
		overallScore += float32(clusterScore+namespaceScore) / float32(maxPermissionScore)
	}

	if overallScore > 0.0 {
		return &storage.Risk_Result{
			Name:    rbacConfigurationHeading,
			Factors: factors,
			Score:   overallScore,
		}
	}

	return nil
}

func (c *saPermissionsMultiplier) getClusterPermissionsScore(ctx context.Context, deployment *storage.Deployment, subject *storage.Subject) (score float32, verbList string, isAdmin bool) {
	clusterEvaluator := utils.NewClusterPermissionEvaluator(deployment.GetClusterId(), c.roleStore, c.bindingStore)

	if clusterEvaluator.IsClusterAdmin(ctx, subject) {
		// maxScore
		clusterReadScore := float32(maxResources) * float32(clusterReadWeight) * float32(k8srbac.ReadResourceVerbs.Cardinality())
		clusterWriteScore := float32(maxResources) * float32(clusterWriteWeight) * float32(k8srbac.WriteResourceVerbs.Cardinality())
		return clusterReadScore + clusterWriteScore, "", true
	}

	permissions := clusterEvaluator.ForSubject(ctx, subject).GetPermissionMap()
	score, verbs := scoreVerbs(permissions, float32(clusterReadWeight), float32(clusterWriteWeight))

	return score, verbs, false
}

func (c *saPermissionsMultiplier) getNamespacePermissionsScore(ctx context.Context, deployment *storage.Deployment, subject *storage.Subject) (float32, string) {
	namespaceEvaluator := utils.NewNamespacePermissionEvaluator(deployment.GetClusterId(),
		deployment.GetNamespace(), c.roleStore, c.bindingStore)
	permissions := namespaceEvaluator.ForSubject(ctx, subject).GetPermissionMap()

	return scoreVerbs(permissions, float32(namespaceReadWeight), float32(namespaceWriteWeight))
}

func (c *saPermissionsMultiplier) tokenAutomounted(ctx context.Context, deployment *storage.Deployment) ([]*storage.Risk_Result_Factor, bool) {
	saName := deployment.GetServiceAccount()

	if saName == "" {
		saName = k8srbac.DefaultServiceAccountName
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, deployment.GetClusterId()).
		AddExactMatches(search.Namespace, deployment.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, saName).ProtoQuery()
	serviceAccounts, err := c.serviceAccountStore.SearchRawServiceAccounts(ctx, q)

	if err != nil {
		log.Errorf("error searching for service account %q for deployment %q: %v", saName, deployment.GetName(), err)
		return nil, false
	}
	if len(serviceAccounts) == 0 {
		return nil, false
	}

	sa := serviceAccounts[0]

	// Reference:
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-the-default-service-account-to-access-the-api-server

	if !deployment.AutomountServiceAccountToken && !sa.AutomountToken {
		return nil, false
	}

	var factors []*storage.Risk_Result_Factor

	if deployment.AutomountServiceAccountToken {
		factors = append(factors, &storage.Risk_Result_Factor{
			Message: fmt.Sprintf("Deployment is configured to automatically mount a token for service account %q",
				saName),
		})
	}

	if sa.AutomountToken {
		factors = append(factors, &storage.Risk_Result_Factor{
			Message: fmt.Sprintf("Service account %q is configured to mount a token into the deployment by default", saName),
		})
	}

	return factors, true
}

func scoreVerbs(permissions map[string]set.StringSet, readWeight float32, writeWeight float32) (float32, string) {
	score := float32(0)
	verbs := make([]string, 0, len(permissions))
	for verb, resources := range permissions {
		// * verb is read write access, so we use the writeWeight
		if verb == "*" {
			score = score + (float32(k8srbac.ResourceVerbs.Cardinality()) * float32(resources.Cardinality()) * writeWeight)
		}

		if k8srbac.ReadResourceVerbs.Contains(verb) {
			if resources.Contains("*") {
				score = score + (float32(maxResources) * readWeight)
			} else {
				score = score + (float32(resources.Cardinality()) * readWeight)
			}
		}

		if k8srbac.WriteResourceVerbs.Contains(verb) {
			if resources.Contains("*") {
				score = score + (float32(maxResources) * writeWeight)
			} else {
				score = score + (float32(resources.Cardinality()) * writeWeight)
			}
		}
		verbs = append(verbs, verb)
	}

	sort.SliceStable(verbs,
		func(i, j int) bool {
			return verbs[i] < verbs[j]
		})

	return score, strings.Join(verbs, ", ")
}
