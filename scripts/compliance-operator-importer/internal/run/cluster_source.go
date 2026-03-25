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
	Label        string // kubeconfig path or context name, for logging
	COClient     cofetch.COClient
	ACSClusterID string
}

// BuildClusterSources creates ClusterSource entries from the config.
//
// Logic:
// - If no multi-cluster flags: single-cluster mode using current context and cfg.ACSClusterID.
// - If --kubeconfig flags: one source per kubeconfig file, discover cluster ID.
// - If --kubecontext flags: one source per context (or all contexts if "all"), discover cluster ID.
// - Manual overrides from --cluster apply to matched contexts.
func BuildClusterSources(ctx context.Context, cfg *models.Config, acsClient models.ACSClient) ([]ClusterSource, error) {
	isMultiClusterMode := len(cfg.Kubeconfigs) > 0 || len(cfg.Kubecontexts) > 0

	if !isMultiClusterMode {
		// Single-cluster mode with auto-discovery.
		coClient, err := cofetch.NewClient(cfg)
		if err != nil {
			return nil, fmt.Errorf("create CO client: %w", err)
		}

		clusterID := cfg.ACSClusterID
		if clusterID == "" {
			// Auto-discover using default kubeconfig context.
			dynClient, err := buildDynamicClientForContext("")
			if err != nil {
				return nil, fmt.Errorf("build dynamic client for current context: %w", err)
			}
			clusterID, err = discover.DiscoverClusterID(ctx, discover.NewK8sDiscoveryClient(dynClient), acsClient, "")
			if err != nil {
				return nil, fmt.Errorf("discover cluster ID for current context: %w", err)
			}
		}

		return []ClusterSource{{
			Label:        "current-context",
			COClient:     coClient,
			ACSClusterID: clusterID,
		}}, nil
	}

	// Parse manual cluster overrides into a map: contextName -> acsClusterName.
	overrides, err := parseClusterOverrides(cfg.ClusterOverrides)
	if err != nil {
		return nil, err
	}

	var sources []ClusterSource

	// Handle --kubeconfig mode.
	if len(cfg.Kubeconfigs) > 0 {
		for _, kubeconfigPath := range cfg.Kubeconfigs {
			coClient, err := cofetch.NewClientForKubeconfig(kubeconfigPath, cfg.CONamespace, cfg.COAllNamespaces)
			if err != nil {
				return nil, fmt.Errorf("create CO client for kubeconfig %q: %w", kubeconfigPath, err)
			}

			// Build dynamic client for discovery.
			dynClient, err := buildDynamicClientForKubeconfig(kubeconfigPath)
			if err != nil {
				return nil, fmt.Errorf("build dynamic client for kubeconfig %q: %w", kubeconfigPath, err)
			}

			// Check for manual override (match by kubeconfig path? Not practical. Skip for kubeconfig mode).
			acsClusterID, err := discover.DiscoverClusterID(ctx, discover.NewK8sDiscoveryClient(dynClient), acsClient, "")
			if err != nil {
				return nil, fmt.Errorf("discover cluster ID for kubeconfig %q: %w", kubeconfigPath, err)
			}

			sources = append(sources, ClusterSource{
				Label:        kubeconfigPath,
				COClient:     coClient,
				ACSClusterID: acsClusterID,
			})
		}
		return sources, nil
	}

	// Handle --kubecontext mode.
	if len(cfg.Kubecontexts) > 0 {
		contexts := cfg.Kubecontexts
		if len(contexts) == 1 && contexts[0] == "all" {
			// Expand "all" to all contexts in the active kubeconfig.
			allContexts, err := listAllContexts()
			if err != nil {
				return nil, fmt.Errorf("list all contexts: %w", err)
			}
			contexts = allContexts
		}

		for _, contextName := range contexts {
			coClient, err := cofetch.NewClientForContext(contextName, cfg.CONamespace, cfg.COAllNamespaces)
			if err != nil {
				return nil, fmt.Errorf("create CO client for context %q: %w", contextName, err)
			}

			// Build dynamic client for discovery.
			dynClient, err := buildDynamicClientForContext(contextName)
			if err != nil {
				return nil, fmt.Errorf("build dynamic client for context %q: %w", contextName, err)
			}

			// Check for manual override.
			manualName := overrides[contextName]
			acsClusterID, err := discover.DiscoverClusterID(ctx, discover.NewK8sDiscoveryClient(dynClient), acsClient, manualName)
			if err != nil {
				return nil, fmt.Errorf("discover cluster ID for context %q: %w", contextName, err)
			}

			sources = append(sources, ClusterSource{
				Label:        contextName,
				COClient:     coClient,
				ACSClusterID: acsClusterID,
			})
		}
		return sources, nil
	}

	return nil, errors.New("no cluster sources configured")
}

// parseClusterOverrides parses --cluster flags into a map: contextName -> acsClusterName.
// Format: ctx=acs-name
func parseClusterOverrides(overrides []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, override := range overrides {
		parts := splitOnce(override, "=")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid --cluster format %q: expected ctx=acs-name", override)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

// splitOnce splits s on the first occurrence of sep.
func splitOnce(s, sep string) []string {
	idx := -1
	for i := 0; i < len(s); i++ {
		if s[i:i+len(sep)] == sep {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

// listAllContexts returns all context names from the active kubeconfig.
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

// buildDynamicClientForKubeconfig creates a dynamic k8s client for the given kubeconfig file.
func buildDynamicClientForKubeconfig(kubeconfigPath string) (dynamic.Interface, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	restConfig, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build rest config: %w", err)
	}
	return dynamic.NewForConfig(restConfig)
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
