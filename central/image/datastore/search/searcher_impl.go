package search

import (
	"context"

	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	"github.com/stackrox/rox/central/cve/edgefields"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	"github.com/stackrox/rox/central/dackbox"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/index"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	imageComponentEdgeSAC "github.com/stackrox/rox/central/imagecomponentedge/sac"
	imageCVEEdgeMappings "github.com/stackrox/rox/central/imagecveedge/mappings"
	imageCVEEdgeSAC "github.com/stackrox/rox/central/imagecveedge/sac"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/derivedfields/counter"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/compound"
	"github.com/stackrox/rox/pkg/search/derivedfields"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.LastUpdatedTime.String(),
	}
	componentOptionsMap = search.CombineOptionsMaps(componentMappings.OptionsMap)
	imageOnlyOptionsMap = search.Difference(
		imageMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentOptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			imageCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)
	deploymentOnlyOptionsMap = search.Difference(deployments.OptionsMap, imageOnlyOptionsMap)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage       store.Store
	indexer       index.Indexer
	graphProvider graph.Provider
	searcher      search.Searcher
}

// SearchImages retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	images, results, err := ds.searchImages(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(images))
	for i, image := range images {
		protoResults = append(protoResults, convertImage(image, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error) {
	images, _, err := ds.searchImages(ctx, q)
	listImages := make([]*storage.ListImage, 0, len(images))
	for _, image := range images {
		listImages = append(listImages, types.ConvertImageToListImage(image))
	}
	return listImages, err
}

// SearchRawImages retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	images, _, err := ds.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	return images, nil
}

func (ds *searcherImpl) searchImages(ctx context.Context, q *v1.Query) ([]*storage.Image, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	var images []*storage.Image
	var newResults []search.Result
	for _, result := range results {
		image, exists, err := ds.storage.GetImageMetadata(ctx, result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		images = append(images, image)
		newResults = append(newResults, result)
	}
	return images, newResults, nil
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Search(inner, q)
	})
	return res, err
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		count, err = ds.searcher.Count(inner, q)
	})
	return count, err
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(graphProvider graph.Provider,
	cveIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	imageComponentEdgeIndexer blevesearch.UnsafeSearcher,
	imageIndexer blevesearch.UnsafeSearcher,
	deploymentIndexer blevesearch.UnsafeSearcher,
	imageCVEEdgeIndexer blevesearch.UnsafeSearcher) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := filtered.UnsafeSearcher(imageComponentEdgeIndexer, imageComponentEdgeSAC.GetSACFilter())
	imageSearcher := filtered.UnsafeSearcher(imageIndexer, imageSAC.GetSACFilter())
	deploymentSearcher := filtered.UnsafeSearcher(deploymentIndexer, deploymentSAC.GetSACFilter())
	imageCVEEdgeSearcher := filtered.UnsafeSearcher(imageCVEEdgeIndexer, imageCVEEdgeSAC.GetSACFilter())

	compoundSearcher := getCompoundImageSearcher(
		cveSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		imageComponentEdgeSearcher,
		imageSearcher,
		deploymentSearcher,
		imageCVEEdgeSearcher,
	)
	filteredSearcher := filtered.Searcher(edgefields.HandleCVEEdgeSearchQuery(compoundSearcher), imageSAC.GetSACFilter())
	// To transform Image to Image Registry, Image Remote, and Image Tag.
	transformedSortSearcher := sortfields.TransformSortFields(filteredSearcher, imageMappings.OptionsMap)
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, transformedSortSearcher)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func getCompoundImageSearcher(
	cveSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	imageComponentEdgeSearcher search.Searcher,
	imageSearcher search.Searcher,
	deploymentSearcher search.Searcher,
	imageCVEEdgeSearcher search.Searcher,
) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher:       scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_IMAGES],
			Options:        cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_IMAGES],
			Options:        imageCVEEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_IMAGE_VULN_EDGE],
		},
		{
			Searcher:       scoped.WithScoping(componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_COMPONENT_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGES],
			Options:        componentCVEEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_COMPONENT_VULN_EDGE],
		},
		{
			Searcher:       scoped.WithScoping(componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_IMAGES],
			Options:        componentOptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
		},
		{
			Searcher:       scoped.WithScoping(imageComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_IMAGES],
			Options:        imageComponentEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_IMAGE_COMPONENT_EDGE],
		},
		{
			IsDefault:  true,
			Searcher:   scoped.WithScoping(imageSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGES)),
			Options:    imageOnlyOptionsMap,
			LinkToPrev: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_IMAGES],
		},
		{
			Searcher:       scoped.WithScoping(deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_DEPLOYMENTS][v1.SearchCategory_IMAGES],
			Options:        deploymentOnlyOptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_DEPLOYMENTS],
		},
	})
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.DeploymentCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ImageToDeploymentPath, deploymentSAC.GetSACFilter()),
	})
}
