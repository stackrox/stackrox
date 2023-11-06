package booleanpolicy

import (
	"errors"
	"time"

	"github.com/stackrox/rox/pkg/booleanpolicy/celcompile"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/negateregocompile"
	"github.com/stackrox/rox/pkg/booleanpolicy/newregocompile"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/regocompile"
	"github.com/stackrox/rox/pkg/features"
)

type factoryWrapper struct {
	legacyFactory         evaluator.Factory
	opaBasedFactory       regocompile.RegoCompiler
	opaOrBasedFactory     newregocompile.RegoCompiler
	opaNegateBasedFactory negateregocompile.RegoCompiler
	celBasedFactory       celcompile.CelCompiler
	// jmespathFactory jmespathcompile.JMESPathCompiler
}

const (
	regoBased         = "rego"
	regoOrBased       = "rego_or_based"
	regoNegateOrBased = "rego_negate_or_based"
	celBased          = "cel"
	jmespathBased     = "jmespath"
)

var (
	legacyCounter = NewDurationCounter(time.Minute, "legacy 1 minute")
	otherCounter  = NewDurationCounter(time.Minute, "other evals 1 minute")
)

type evaluatorWrapper struct {
	otherEvaluators map[string]evaluator.Evaluator
	legacyEvaluator evaluator.Evaluator

	q *query.Query
}

func (e *evaluatorWrapper) Evaluate(obj *pathutil.AugmentedObj) (*evaluator.Result, bool) {
	for name := range e.otherEvaluators {
		evaluator := e.otherEvaluators[name]
		result, matched := evaluator.Evaluate(obj)
		otherCounter.Add()
		return result, matched
	}

	legacyResult, legacyMatched := e.legacyEvaluator.Evaluate(obj)
	legacyCounter.Add()

	return legacyResult, legacyMatched
}

func (f *factoryWrapper) GenerateEvaluator(q *query.Query) (evaluator.Evaluator, error) {
	e := &evaluatorWrapper{q: q, otherEvaluators: make(map[string]evaluator.Evaluator)}
	start := time.Now()
	legacyEvaluator, err := f.legacyFactory.GenerateEvaluator(q)
	if err != nil {
		return nil, err
	}
	legacyDuration := time.Since(start).Nanoseconds()
	log.Debugf("Legacy compile %d", legacyDuration)

	e.legacyEvaluator = legacyEvaluator
	if features.PolicyEngineEvaluatorTest.Enabled() {
		if f.opaBasedFactory != nil {
			start := time.Now()
			regoEvaluator, err := f.opaBasedFactory.CompileRegoBasedEvaluator(q)
			if err != nil {
				if !errors.Is(err, regocompile.ErrRegoNotYetSupported) {
					return nil, err
				}
			} else {
				e.otherEvaluators[regoBased] = regoEvaluator
				duration := time.Since(start).Nanoseconds()
				log.Infof("Rego base compile %d, which is %.2f times of legacy", duration, float64(duration)/float64(legacyDuration))
			}
		}
		if f.opaOrBasedFactory != nil {
			start := time.Now()
			regoOrEvaluator, err := f.opaOrBasedFactory.CompileRegoBasedEvaluator(q)
			if err != nil {
				if !errors.Is(err, newregocompile.ErrRegoNotYetSupported) {
					return nil, err
				}
			} else {
				e.otherEvaluators[regoOrBased] = regoOrEvaluator
				duration := time.Since(start).Nanoseconds()
				log.Infof("Rego or compile %d, which is %.2f times of legacy", duration, float64(duration)/float64(legacyDuration))
			}
		}

		if f.opaNegateBasedFactory != nil {
			start := time.Now()
			regoNegateEvaluator, err := f.opaNegateBasedFactory.CompileRegoBasedEvaluator(q)
			if err != nil {
				if !errors.Is(err, negateregocompile.ErrRegoNotYetSupported) {
					return nil, err
				}
			} else {
				e.otherEvaluators[regoNegateOrBased] = regoNegateEvaluator
				duration := time.Since(start).Nanoseconds()
				log.Infof("Rego negate compile %d, which is %.2f times of legacy", duration, float64(duration)/float64(legacyDuration))
			}
		}

		if f.celBasedFactory != nil {
			start := time.Now()
			celEvaluator, err := f.celBasedFactory.CompileCelBasedEvaluator(q)
			if err != nil {
				if !errors.Is(err, celcompile.ErrCelNotYetSupported) {
					return nil, err
				}
			} else {
				e.otherEvaluators[celBased] = celEvaluator
				duration := time.Since(start).Nanoseconds()
				log.Infof("CEL compile %d, which is %.2f times of legacy", duration, float64(duration)/float64(legacyDuration))
			}
		}
	}

	return e, nil
}

// MustCreateFactoryWrapper returns a factory wrapper.
// A factory wrapper routes between the OPA and the legacy factory
// depending on the value of the feature flag.
// This is temporary code until the OPA feature flag is removed.
func MustCreateFactoryWrapper(objMeta *pathutil.AugmentedObjMeta) evaluator.Factory {
	evaluators := getConfiguredEvaluatorTypes()

	fw := &factoryWrapper{
		legacyFactory: evaluator.MustCreateNewFactory(objMeta),
	}
	if evaluators.Contains(RegoBase) {
		fw.opaBasedFactory = regocompile.MustCreateRegoCompiler(objMeta)
	}
	if evaluators.Contains(RegoOr) {
		fw.opaOrBasedFactory = newregocompile.MustCreateRegoCompiler(objMeta)
	}
	if evaluators.Contains(RegoNegate) {
		fw.opaNegateBasedFactory = negateregocompile.MustCreateRegoCompiler(objMeta)
	}
	if evaluators.Contains(Cel) {
		fw.celBasedFactory = celcompile.MustCreateCompiler(objMeta)
	}
	return fw
}
