package run

import (
	"context"
	"errors"
	"fmt"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/discover"
	"github.com/stackrox/co-acs-importer/internal/models"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterSource represents a single source cluster with its CO client and ACS cluster ID.
type ClusterSource struct {
	Label        string // context name, for logging
	COClient     cofetch.COClient
	ACSClusterID string
}

// BuildClusterSources creates ClusterSource entries by iterating all contexts
// in the merged kubeconfig. If cfg.Contexts is non-empty, only those contexts
// are used.
func BuildClusterSources(ctx context.Context, cfg *models.Config, acsClient models.ACSClient) ([]ClusterSource, error) {
	allContexts, err := listAllContexts()
	if err != nil {
		return nil, err
	}

	contexts := allContexts
	if len(cfg.Contexts) > 0 {
		contexts = filterContexts(allContexts, cfg.Contexts)
		if len(contexts) == 0 {
			return nil, fmt.Errorf("none of the requested --context values match available contexts %v", allContexts)
		}
	}

	var sources []ClusterSource
	for _, contextName := range contexts {
		coClient, err := cofetch.NewClientForContext(contextName, cfg.CONamespace, cfg.COAllNamespaces)
		if err != nil {
			return nil, fmt.Errorf("create CO client for context %q: %w", contextName, err)
		}

		dynClient, err := buildDynamicClientForContext(contextName)
		if err != nil {
			return nil, fmt.Errorf("build dynamic client for context %q: %w", contextName, err)
		}

		acsClusterID, err := discover.DiscoverClusterID(ctx, discover.NewK8sDiscoveryClient(dynClient), acsClient)
		if err != nil {
			return nil, fmt.Errorf("discover cluster ID for context %q: %w", contextName, err)
		}

		sources = append(sources, ClusterSource{
			Label:        contextName,
			COClient:     coClient,
			ACSClusterID: acsClusterID,
		})
	}

	if len(sources) == 0 {
		return nil, errors.New("no contexts found in kubeconfig")
	}
	return sources, nil
}

// filterContexts returns only contexts whose names appear in the wanted set.
func filterContexts(all []string, wanted []string) []string {
	set := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		set[w] = true
	}
	var result []string
	for _, c := range all {
		if set[c] {
			result = append(result, c)
		}
	}
	return result
}

// listAllContexts returns all context names from the merged kubeconfig.
func listAllContexts() ([]string, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := loadingRules.Load()
	if err != nil {
		return nil, fmt.Errorf("load kubeconfig: %w", err)
	}

	var contexts []string
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	if len(contexts) == 0 {
		return nil, errors.New("no contexts found in kubeconfig")
	}
	return contexts, nil
}

// buildDynamicClientForContext creates a dynamic k8s client for the given context.
func buildDynamicClientForContext(contextName string) (dynamic.Interface, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}
	return dynamic.NewForConfig(restConfig)
}
