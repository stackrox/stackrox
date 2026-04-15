package registry

import (
	"sort"

	"github.com/stackrox/rox/central/risk/scorer/plugin"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// Registry manages risk scoring plugin configurations.
// It provides thread-safe access to enabled plugins sorted by priority.
type Registry interface {
	// Register adds a plugin implementation that can be configured.
	Register(p plugin.Plugin)

	// UpsertConfig adds or updates a plugin configuration.
	// The plugin must be registered before it can be configured.
	UpsertConfig(config *plugin.Config) error

	// DeleteConfig removes a plugin configuration.
	DeleteConfig(id string) error

	// GetEnabledPlugins returns all enabled plugins sorted by priority.
	GetEnabledPlugins() []*plugin.ConfiguredPlugin

	// GetConfig returns the configuration for a specific plugin.
	GetConfig(id string) (*plugin.Config, bool)
}

// New creates a new plugin registry.
func New() Registry {
	return &registryImpl{
		plugins: make(map[string]plugin.Plugin),
		configs: make(map[string]*plugin.Config),
	}
}

type registryImpl struct {
	mu      sync.RWMutex
	plugins map[string]plugin.Plugin  // plugin name -> implementation
	configs map[string]*plugin.Config // config ID -> config
}

func (r *registryImpl) Register(p plugin.Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins[p.Name()] = p
	log.Infof("Registered risk scoring plugin: %s", p.Name())
}

func (r *registryImpl) UpsertConfig(config *plugin.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if config.Type == plugin.PluginTypeBuiltin {
		if _, ok := r.plugins[config.Name]; !ok {
			log.Warnf("No registered plugin for config %s (plugin name: %s)", config.ID, config.Name)
		}
	}

	r.configs[config.ID] = config
	log.Infof("Upserted plugin config: %s (enabled: %v, weight: %.2f, priority: %d)",
		config.ID, config.Enabled, config.Weight, config.Priority)
	return nil
}

func (r *registryImpl) DeleteConfig(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.configs, id)
	log.Infof("Deleted plugin config: %s", id)
	return nil
}

func (r *registryImpl) GetEnabledPlugins() []*plugin.ConfiguredPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*plugin.ConfiguredPlugin
	for _, config := range r.configs {
		if !config.Enabled {
			continue
		}

		p, ok := r.plugins[config.Name]
		if !ok {
			log.Warnf("Config %s references unknown plugin %s", config.ID, config.Name)
			continue
		}

		result = append(result, &plugin.ConfiguredPlugin{
			Plugin: p,
			Config: config,
		})
	}

	// Sort by priority (lower = earlier)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Config.Priority < result[j].Config.Priority
	})

	return result
}

func (r *registryImpl) GetConfig(id string) (*plugin.Config, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config, ok := r.configs[id]
	return config, ok
}

// Singleton returns the global plugin registry.
func Singleton() Registry {
	return singletonInstance
}

var singletonInstance = New()
