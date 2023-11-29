package manager

import (
	"context"
	"errors"
	"time"

	"github.com/stackrox/rox/central/compliance/data"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/central/scrape/factory"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
)

type dataPromise interface {
	WaitForResult(cancel concurrency.Waitable) (framework.ComplianceDataRepository, error)
}

type fixedDataPromise struct {
	dataRepo framework.ComplianceDataRepository
	err      error
}

func newFixedDataPromise(ctx context.Context, dataRepoFactory data.RepositoryFactory, domain framework.ComplianceDomain) dataPromise {
	dataRepo, err := dataRepoFactory.CreateDataRepository(ctx, domain, nil)

	return &fixedDataPromise{
		dataRepo: dataRepo,
		err:      err,
	}
}

func (p *fixedDataPromise) WaitForResult(_ concurrency.Waitable) (framework.ComplianceDataRepository, error) {
	return p.dataRepo, p.err
}

// scrapePromise allows access to compliance data in an asynchronous way, allowing multiple runs to wait on the same
// scrape process.
type scrapePromise struct {
	domain          framework.ComplianceDomain
	dataRepoFactory data.RepositoryFactory

	finishedSig concurrency.ErrorSignal

	result framework.ComplianceDataRepository
}

// createAndRunScrape creates and returns a scrapePromise for the given domain. The returned promise will be running.
func createAndRunScrape(ctx context.Context, scrapeFactory factory.ScrapeFactory, dataRepoFactory data.RepositoryFactory, domain framework.ComplianceDomain, timeout time.Duration, standardIDs []string) *scrapePromise {
	promise := &scrapePromise{
		domain:          domain,
		finishedSig:     concurrency.NewErrorSignal(),
		dataRepoFactory: dataRepoFactory,
	}
	go promise.run(ctx, scrapeFactory, domain, timeout, standardIDs)
	return promise
}

func (p *scrapePromise) finish(ctx context.Context, scrapeResult map[string]*compliance.ComplianceReturn, err error) {
	if err != nil {
		log.Errorf("Scrape failed: %v. Using partial data from %d/%d nodes", err, len(scrapeResult), len(p.domain.Nodes()))
	}
	if len(scrapeResult) != len(p.domain.Nodes()) {
		var missingNodes []string
		for _, n := range p.domain.Nodes() {
			if _, ok := scrapeResult[n.Node().GetName()]; !ok {
				missingNodes = append(missingNodes, n.Node().GetName())
			}
		}
		log.Warnf("Did not collect scrape data for %+v", missingNodes)
	}

	log.Info("CreateDataRepository finish")
	p.result, err = p.dataRepoFactory.CreateDataRepository(ctx, p.domain)
	if err == nil {
		p.result.AddHostScrapedData(scrapeResult)
	}
	p.finishedSig.SignalWithError(err)
}

func (p *scrapePromise) WaitForResult(cancel concurrency.Waitable) (framework.ComplianceDataRepository, error) {
	err, done := p.finishedSig.WaitUntil(cancel)
	if !done {
		return nil, errors.New("cancelled while waiting for compliance results")
	}
	if err != nil {
		return nil, err
	}
	return p.result, nil
}

func (p *scrapePromise) run(ctx context.Context, scrapeFactory factory.ScrapeFactory, domain framework.ComplianceDomain, timeout time.Duration, standardIDs []string) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	scrapeResults, err := scrapeFactory.RunScrape(domain, ctx, standardIDs)
	p.finish(ctx, scrapeResults, err)
}
