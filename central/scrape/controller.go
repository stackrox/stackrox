package scrape

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/scrape/sensor/accept"
	"github.com/stackrox/rox/central/scrape/sensor/emit"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Factory starts and stops scrapes.
type Factory interface {
	RunScrape(domain framework.ComplianceDomain, kill concurrency.Waitable) (map[string]*compliance.ComplianceReturn, error)
}

// NewFactory returns a new instance of a Factory.
func NewFactory(emitter emit.Emitter, accepter accept.Accepter) Factory {
	return &controllerImpl{
		emitter:  emitter,
		accepter: accepter,
	}
}

//go:generate mockgen-wrapper Factory
