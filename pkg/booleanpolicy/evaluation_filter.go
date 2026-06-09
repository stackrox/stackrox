package booleanpolicy

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/utils"
)

// MatcherKind identifies which Match* function is being compiled.
// Used as a bitmask so a FilterPlugin can declare exactly which matchers it applies to.
type MatcherKind uint8

const (
	DeploymentKind  MatcherKind = 1 << iota // MatchDeployment
	ImageKind                               // MatchImage (build-time)
	ProcessKind                             // MatchDeploymentWithProcess
	NetworkFlowKind                         // MatchDeploymentWithNetworkFlow
	FileAccessKind                          // MatchDeploymentWithFileAccess
	KubeEventKind                           // MatchKubeEvent
	AuditLogKind                            // MatchAuditLogEvent
	NodeKind                                // MatchNodeWithFileAccess
)

// ContextExtractor extracts the data a plugin needs from the raw match context.
// Each MatcherKind has its own extractor because different matchers provide different contexts.
// Returning nil signals that no data is available and no filtering should occur.
type ContextExtractor func(matchData interface{}) interface{}

// ValueFilterFactory is called once per Match* invocation with data already extracted
// by the ContextExtractor for this kind. Returns nil if no filtering is needed.
type ValueFilterFactory func(data interface{}) pathutil.ValueFilter

// FilterPlugin compiles the relevant fields from the EvaluationFilter at policy
// build time into a ValueFilterFactory. Returning nil means the plugin does not
// apply for this particular filter configuration.
type FilterPlugin func(f *storage.EvaluationFilter) ValueFilterFactory

type registeredPlugin struct {
	plugin     FilterPlugin
	extractors map[MatcherKind]ContextExtractor
}

// evaluationFilterType caches the reflect.Type of EvaluationFilter for field iteration.
var evaluationFilterType = reflect.TypeOf(storage.EvaluationFilter{})

// filterPluginRegistry maps each EvaluationFilter field type to its owning plugin.
var filterPluginRegistry = map[reflect.Type]registeredPlugin{}

// RegisterFilterPlugin registers a plugin for the EvaluationFilter field of type T,
// with per-kind extractors that extract the data the plugin needs from each matcher context.
// Panics if T is not a field of EvaluationFilter.
// Not thread-safe; call only from init() functions.
func RegisterFilterPlugin[T any](plugin FilterPlugin, extractors map[MatcherKind]ContextExtractor) {
	key := reflect.TypeOf((*T)(nil)).Elem()
	for i := 0; i < evaluationFilterType.NumField(); i++ {
		if evaluationFilterType.Field(i).Type == key {
			filterPluginRegistry[key] = registeredPlugin{plugin: plugin, extractors: extractors}
			return
		}
	}
	utils.CrashOnError(errors.Errorf("RegisterFilterPlugin: %v is not a field type of storage.EvaluationFilter", key))
}

type compiledFactory struct {
	factory    ValueFilterFactory
	extractors map[MatcherKind]ContextExtractor
}

// CompiledEvaluationFilter holds the active factories produced from the EvaluationFilter.
type CompiledEvaluationFilter []compiledFactory

// CompileEvaluationFilter iterates the fields of the EvaluationFilter and invokes the plugin registered
// for each field type.
func CompileEvaluationFilter(f *storage.EvaluationFilter) CompiledEvaluationFilter {
	if f == nil {
		return nil
	}
	var result CompiledEvaluationFilter
	for i := 0; i < evaluationFilterType.NumField(); i++ {
		p, ok := filterPluginRegistry[evaluationFilterType.Field(i).Type]
		if !ok {
			continue
		}
		if factory := p.plugin(f); factory != nil {
			result = append(result, compiledFactory{factory: factory, extractors: p.extractors})
		}
	}
	return result
}

// ForKind returns ValueFilterFactories for the given matcher kind, each pre-wrapped
// with its kind-specific ContextExtractor. Called once per Build*Matcher invocation.
func (c CompiledEvaluationFilter) ForKind(kind MatcherKind) []ValueFilterFactory {
	var factories []ValueFilterFactory
	for _, cf := range c {
		extractor, ok := cf.extractors[kind]
		if !ok {
			continue
		}
		factory, extract := cf.factory, extractor
		factories = append(factories, func(matchData interface{}) pathutil.ValueFilter {
			return factory(extract(matchData))
		})
	}
	return factories
}

// combineValueFilters calls each factory with matchData and returns a combined ValueFilter.
// All factory results are AND-ed: an element must pass every active filter.
// matchData is the raw match context; each factory extracts what it needs via its extractor.
// Returns nil when no factories produce a non-nil filter.
func combineValueFilters(factories []ValueFilterFactory, matchData interface{}) pathutil.ValueFilter {
	if len(factories) == 0 {
		return nil
	}
	active := make([]pathutil.ValueFilter, 0, len(factories))
	for _, factory := range factories {
		if f := factory(matchData); f != nil {
			active = append(active, f)
		}
	}
	switch len(active) {
	case 0:
		return nil
	case 1:
		return active[0]
	default:
		return func(v pathutil.AugmentedValue, i int) bool {
			for _, f := range active {
				if !f(v, i) {
					return false
				}
			}
			return true
		}
	}
}
