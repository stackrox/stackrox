package multipliers

import (
	"fmt"

	"github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

const (
	// RBACConfigurationHeading is the risk result name for scores calculated by this multiplier.
	RBACConfigurationHeading = "RBAC Configuration"
)

// type secretAutomountMultiplier is a scorer for an auto mounted secret in the deployment.
type secretAutomountMultiplier struct {
	serviceAccountStore datastore.DataStore
}

// NewSecretAutomount provides a multiplier that scores the deployment based on the existence of an auto mounted secret token
func NewSecretAutomount(store datastore.DataStore) Multiplier {
	return &secretAutomountMultiplier{
		serviceAccountStore: store,
	}
}

// Score takes a deployment and evaluates its risk based on auto mounted secret token
func (c *secretAutomountMultiplier) Score(deployment *storage.Deployment) *storage.Risk_Result {
	saName := deployment.GetServiceAccount()

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, deployment.GetClusterId()).
		AddExactMatches(search.Namespace, deployment.GetNamespace()).
		AddExactMatches(search.ServiceAccountName, saName).ProtoQuery()
	serviceAccounts, err := c.serviceAccountStore.SearchRawServiceAccounts(q)

	if err != nil {
		log.Errorf("error searching for service account %q: %v", saName, err)
	}
	if len(serviceAccounts) == 0 {
		log.Errorf("could not find service account %q: %v", saName, err)
		return nil
	}

	sa := serviceAccounts[0]

	// Reference:
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-the-default-service-account-to-access-the-api-server

	if !deployment.AutomountServiceAccountToken && !sa.AutomountToken {
		return nil
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

	return &storage.Risk_Result{
		Name:    RBACConfigurationHeading,
		Factors: factors,
		Score:   2,
	}
}
