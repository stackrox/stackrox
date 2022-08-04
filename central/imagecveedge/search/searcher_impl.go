package search

import (
	"context"

	clusterIndexer "github.com/stackrox/rox/central/cluster/index"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	clusterSAC "github.com/stackrox/rox/central/cluster/sac"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	"github.com/stackrox/rox/central/dackbox"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	imageComponentEdgeSAC "github.com/stackrox/rox/central/imagecomponentedge/sac"
	imageCVEEdgeIndexer "github.com/stackrox/rox/central/imagecveedge/index"
	imageCVEEdgeMappings "github.com/stackrox/rox/central/imagecveedge/mappings"
	imageCVEEdgeSAC "github.com/stackrox/rox/central/imagecveedge/sac"
	"github.com/stackrox/rox/central/imagecveedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/compound"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped"
)

type searcherImpl struct {
	storage  store.Store
	indexer  imageCVEEdgeIndexer.Indexer
	searcher search.Searcher
}

// SearchEdges returns the search results from indexed image-cve edges for the query.
func (ds *searcherImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

// SearchRawEdges retrieves image-cve edges from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error) {
	return ds.searchImageCVEEdges(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// resultsToImageCVEEdges returns the ImageCVEEdges from the db for the given search results.
func (ds *searcherImpl) resultsToImageCVEEdges(ctx context.Context, results []search.Result) ([]*storage.ImageCVEEdge, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	edges, missingIndices, err := ds.resultsToImageCVEEdges(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(edges, results), nil
}

func (ds *searcherImpl) searchImageCVEEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(cveIndexer cveIndexer.Indexer,
	imageCVEEdgeIndexer imageCVEEdgeIndexer.Indexer,
	componentCVEEdgeIndexer componentCVEEdgeIndexer.Indexer,
	componentIndexer componentIndexer.Indexer,
	imageComponentEdgeIndexer imageComponentEdgeIndexer.Indexer,
	imageIndexer imageIndexer.Indexer,
	deploymentIndexer deploymentIndexer.Indexer,
	clusterIndexer clusterIndexer.Indexer) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	imageCVEEdgeSearcher := filtered.UnsafeSearcher(imageCVEEdgeIndexer, imageCVEEdgeSAC.GetSACFilter())
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := filtered.UnsafeSearcher(imageComponentEdgeIndexer, imageComponentEdgeSAC.GetSACFilter())
	imageSearcher := filtered.UnsafeSearcher(imageIndexer, imageSAC.GetSACFilter())
	deploymentSearcher := filtered.UnsafeSearcher(deploymentIndexer, deploymentSAC.GetSACFilter())
	clusterSearcher := filtered.UnsafeSearcher(clusterIndexer, clusterSAC.GetSACFilter())

	compoundSearcher := getCompoundImageCVESearcher(compoundImageCVESearcherArg{
		cveSearcher:                cveSearcher,
		imageCVEEdgeSearcher:       imageCVEEdgeSearcher,
		componentSearcher:          componentSearcher,
		componentCVEEdgeSearcher:   componentCVEEdgeSearcher,
		imageComponentEdgeSearcher: imageComponentEdgeSearcher,
		imageSearcher:              imageSearcher,
		deploymentSearcher:         deploymentSearcher,
		clusterSearcher:            clusterSearcher,
	})
	filteredSearcher := filtered.Searcher(compoundSearcher, imageCVEEdgeSAC.GetSACFilter())
	return paginated.Paginated(filteredSearcher)
}

type compoundImageCVESearcherArg struct {
	componentCVEEdgeSearcher   search.Searcher
	cveSearcher                search.Searcher
	componentSearcher          search.Searcher
	imageComponentEdgeSearcher search.Searcher
	imageCVEEdgeSearcher       search.Searcher
	imageSearcher              search.Searcher
	deploymentSearcher         search.Searcher
	clusterSearcher            search.Searcher
}

func getCompoundImageCVESearcher(arg compoundImageCVESearcherArg) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher:       scoped.WithScoping(arg.cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        cveMappings.OptionsMap,
		},
		{
			IsDefault: true,
			Searcher:  scoped.WithScoping(arg.imageCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Options:   imageCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        componentCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        componentMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.imageComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        imageComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.imageSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        dackbox.ImageOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_DEPLOYMENTS][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        dackbox.DeploymentOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.clusterSearcher, dackbox.ToCategory(v1.SearchCategory_CLUSTERS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_CLUSTERS][v1.SearchCategory_IMAGE_VULN_EDGE],
			Options:        clusterMappings.OptionsMap,
		},
	})
}
