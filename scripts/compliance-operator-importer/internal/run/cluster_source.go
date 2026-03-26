package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/discover"
	"github.com/stackrox/co-acs-importer/internal/models"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterSource represents a single source cluster with its CO client and ACS cluster ID.
type ClusterSource struct {
	Label        string // context name, for logging
	COClient     cofetch.COClient
	ACSClusterID string
}

// contextRef pairs a kubeconfig file with one of its contexts.
type contextRef struct {
	Context        string
	KubeconfigFile string
}

// BuildClusterSources creates ClusterSource entries by loading each kubeconfig
// file independently (no merging). If cfg.Contexts is non-empty, only matching
// contexts are used.
func BuildClusterSources(ctx context.Context, cfg *models.Config, acsClient models.ACSClient) ([]ClusterSource, error) {
	allRefs, err := listContextRefs()
	if err != nil {
		return nil, err
	}

	refs := allRefs
	if len(cfg.Contexts) > 0 {
		refs = filterRefs(allRefs, cfg.Contexts)
		if len(refs) == 0 {
			return nil, fmt.Errorf("none of the requested --context values match available contexts %v", contextNames(allRefs))
		}
	}

	var sources []ClusterSource
	for _, ref := range refs {
		restCfg, err := restConfigForRef(ref)
		if err != nil {
			return nil, fmt.Errorf("build rest config for context %q: %w", ref.Context, err)
		}

		coClient, err := cofetch.NewClientFromRestConfig(restCfg, cfg.CONamespace, cfg.COAllNamespaces)
		if err != nil {
			return nil, fmt.Errorf("create CO client for context %q: %w", ref.Context, err)
		}

		dynClient, err := dynamic.NewForConfig(restCfg)
		if err != nil {
			return nil, fmt.Errorf("build dynamic client for context %q: %w", ref.Context, err)
		}

		acsClusterID, err := discover.DiscoverClusterID(ctx, discover.NewK8sDiscoveryClient(dynClient), acsClient)
		if err != nil {
			return nil, fmt.Errorf("discover cluster ID for context %q: %w", ref.Context, err)
		}

		sources = append(sources, ClusterSource{
			Label:        ref.Context,
			COClient:     coClient,
			ACSClusterID: acsClusterID,
		})
	}

	if len(sources) == 0 {
		return nil, errors.New("no contexts found in kubeconfig")
	}
	return sources, nil
}

// filterRefs returns refs whose context name appears in the wanted set.
func filterRefs(all []contextRef, wanted []string) []contextRef {
	set := make(map[string]bool, len(wanted))
	for _, w := range wanted {
		set[w] = true
	}
	var result []contextRef
	for _, r := range all {
		if set[r.Context] {
			result = append(result, r)
		}
	}
	return result
}

func contextNames(refs []contextRef) []string {
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.Context
	}
	return names
}

// listContextRefs enumerates contexts from each kubeconfig file independently.
// Each file is loaded in isolation so that user/cluster entries with the same
// name in different files don't collide.
func listContextRefs() ([]contextRef, error) {
	files := kubeconfigFiles()
	if len(files) == 0 {
		return nil, errors.New("no kubeconfig files found (check KUBECONFIG or ~/.kube/config)")
	}

	var refs []contextRef
	for _, file := range files {
		cfg, err := clientcmd.LoadFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig %q: %w", file, err)
		}
		for ctxName := range cfg.Contexts {
			refs = append(refs, contextRef{Context: ctxName, KubeconfigFile: file})
		}
	}

	if len(refs) == 0 {
		return nil, errors.New("no contexts found in kubeconfig files")
	}
	return refs, nil
}

// kubeconfigFiles returns the list of kubeconfig file paths from the KUBECONFIG
// env var, or falls back to ~/.kube/config.
func kubeconfigFiles() []string {
	env := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if env == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		defaultPath := filepath.Join(home, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName)
		if _, err := os.Stat(defaultPath); err == nil {
			return []string{defaultPath}
		}
		return nil
	}

	parts := filepath.SplitList(env)
	var files []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}
	return files
}

// restConfigForRef builds a rest.Config from a specific kubeconfig file and context,
// without merging with other kubeconfig files.
func restConfigForRef(ref contextRef) (*rest.Config, error) {
	loadingRules := &clientcmd.ClientConfigLoadingRules{
		ExplicitPath: ref.KubeconfigFile,
	}
	overrides := &clientcmd.ConfigOverrides{CurrentContext: ref.Context}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	return kubeConfig.ClientConfig()
}
