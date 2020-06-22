package factory

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
)

// ScrapeFactory allows running scrapes, automatically inferring the cluster from the compliance domain.
type ScrapeFactory interface {
	RunScrape(domain framework.ComplianceDomain, kill concurrency.Waitable, standardIDs []string) (map[string]*compliance.ComplianceReturn, error)
}

//go:generate mockgen-wrapper
