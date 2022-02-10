package booleanpolicy

import (
	"errors"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/booleanpolicy/regocompile"
	"github.com/stackrox/rox/pkg/features"
)

type factoryWrapper struct {
	legacyFactory   evaluator.Factory
	opaBasedFactory regocompile.RegoCompiler
}

type evaluatorWrapper struct {
	regoEvaluator   evaluator.Evaluator
	legacyEvaluator evaluator.Evaluator

	q *query.Query
}

func (e *evaluatorWrapper) Evaluate(obj *pathutil.AugmentedObj) (*evaluator.Result, bool) {
	if e.regoEvaluator == nil {
		return e.legacyEvaluator.Evaluate(obj)
	}
	start := time.Now()
	regoResult, regoMatched := e.regoEvaluator.Evaluate(obj)
	regoDone := time.Now()
	legacyResult, legacyMatched := e.legacyEvaluator.Evaluate(obj)
	legacyDone := time.Now()
	if regoDone.Sub(start) > 10*time.Millisecond || regoMatched != legacyMatched {
		objValue, _ := obj.GetFullValue()
		log.Infof("Rego took %s; legacy took %s", regoDone.Sub(start), legacyDone.Sub(regoDone))
		log.Errorf("OPA matched: %v\n; legacy matched: %v\n;"+
			" opa result %+v\n\n; legacy result: %+v\n\n; query was %s\n\n, obj name is %v",
			regoMatched, legacyMatched, regoResult, legacyResult, spew.Sdump(e.q), objValue["Name"])
	}
	return legacyResult, legacyMatched
}

func (f *factoryWrapper) GenerateEvaluator(q *query.Query) (evaluator.Evaluator, error) {
	e := &evaluatorWrapper{q: q}
	if features.OPABasedEvaluator.Enabled() {
		regoEvaluator, err := f.opaBasedFactory.CompileRegoBasedEvaluator(q)
		if err != nil {
			if !errors.Is(err, regocompile.ErrRegoNotYetSupported) {
				return nil, err
			}
		} else {
			e.regoEvaluator = regoEvaluator
		}
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
		legacyFactory:   evaluator.MustCreateNewFactory(objMeta),
		opaBasedFactory: regocompile.MustCreateRegoCompiler(objMeta),
	}
}
