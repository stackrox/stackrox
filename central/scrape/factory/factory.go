package factory

import (
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/internalapi/compliance"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// ScrapeFactory allows running scrapes, automatically inferring the cluster from the compliance domain.
type ScrapeFactory interface {
	RunScrape(domain framework.ComplianceDomain, kill concurrency.Waitable, standardIDs []string) (map[string]*compliance.ComplianceReturn, error)
}

//go:generate mockgen-wrapper
