package booleanpolicy

import (
	"errors"
	"sort"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/negateregocompile"
	"github.com/stackrox/rox/pkg/booleanpolicy/newregocompile"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/regocompile"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/maputil"
)

type factoryWrapper struct {
	legacyFactory         evaluator.Factory
	opaBasedFactory       regocompile.RegoCompiler
	opaOrBasedFactory     newregocompile.RegoCompiler
	opaNegateBasedFactory negateregocompile.RegoCompiler
	// jmespathFactory jmespathcompile.JMESPathCompiler
}

const (
	regoBased         = "rego"
	regoOrBased       = "rego_or_based"
	regoNegateOrBased = "rego_negate_or_based"
	celBased          = "cel"
	jmespathBased     = "jmespath"
)

type evaluatorWrapper struct {
	otherEvaluators map[string]evaluator.Evaluator
	legacyEvaluator evaluator.Evaluator

	q *query.Query
}

func (e *evaluatorWrapper) Evaluate(obj *pathutil.AugmentedObj) (*evaluator.Result, bool) {
	if len(e.otherEvaluators) == 0 {
		return e.legacyEvaluator.Evaluate(obj)
	}

	start := time.Now()
	legacyResult, legacyMatched := e.legacyEvaluator.Evaluate(obj)
	legacyDuration := time.Now().Sub(start)
	log.Infof("legacy took %d ns", legacyDuration.Nanoseconds())

	keys := maputil.Keys(e.otherEvaluators)
	sort.Strings(keys)

	for _, name := range keys {
		evaluator := e.otherEvaluators[name]
		start = time.Now()
		result, matched := evaluator.Evaluate(obj)
		duration := time.Now().Sub(start)
		log.Infof("%s took %d ns", name, duration.Nanoseconds())

		if duration > 1*time.Millisecond || matched != legacyMatched {
			objValue, _ := obj.GetFullValue()
			log.Errorf("%s matched: %v\n; legacy matched: %v\n;"+
				" %s result %+v\n\n; legacy result: %+v\n\n; query was %s\n\n, obj name is %v",
				name, matched, legacyMatched, name, result, legacyResult, spew.Sdump(e.q), objValue["Name"])
		}
	}
	return legacyResult, legacyMatched
}

func (f *factoryWrapper) GenerateEvaluator(q *query.Query) (evaluator.Evaluator, error) {
	e := &evaluatorWrapper{q: q, otherEvaluators: make(map[string]evaluator.Evaluator)}
	if features.OPABasedEvaluator.Enabled() {
		regoEvaluator, err := f.opaBasedFactory.CompileRegoBasedEvaluator(q)
		if err != nil {
			if !errors.Is(err, regocompile.ErrRegoNotYetSupported) {
				return nil, err
			}
		} else {
			e.otherEvaluators[regoBased] = regoEvaluator
		}
		regoOrEvaluator, err := f.opaOrBasedFactory.CompileRegoBasedEvaluator(q)
		if err != nil {
			if !errors.Is(err, newregocompile.ErrRegoNotYetSupported) {
				return nil, err
			}
		} else {
			e.otherEvaluators[regoOrBased] = regoOrEvaluator
		}
		regoNegateEvaluator, err := f.opaNegateBasedFactory.CompileRegoBasedEvaluator(q)
		if err != nil {
			if !errors.Is(err, negateregocompile.ErrRegoNotYetSupported) {
				return nil, err
			}
		} else {
			e.otherEvaluators[regoNegateOrBased] = regoNegateEvaluator
		}
	}

	if features.JmesPathBasedEvaluator.Enabled() {

	}

	if features.CelBasedEvaluator.Enabled() {

	}

	legacyEvaluator, err := f.legacyFactory.GenerateEvaluator(q)
	if err != nil {
		return nil, err
	}
	e.legacyEvaluator = legacyEvaluator
	return e, nil
}

// MustCreateFactoryWrapper returns a factory wrapper.
// A factory wrapper routes between the OPA and the legacy factory
// depending on the value of the feature flag.
// This is temporary code until the OPA feature flag is removed.
func MustCreateFactoryWrapper(objMeta *pathutil.AugmentedObjMeta) evaluator.Factory {
	return &factoryWrapper{
		legacyFactory:         evaluator.MustCreateNewFactory(objMeta),
		opaBasedFactory:       regocompile.MustCreateRegoCompiler(objMeta),
		opaOrBasedFactory:     newregocompile.MustCreateRegoCompiler(objMeta),
		opaNegateBasedFactory: negateregocompile.MustCreateRegoCompiler(objMeta),
		// jmespathFactory: jmespathcompile.MustCreateJMESPathCompiler(objMeta),
	}
}
