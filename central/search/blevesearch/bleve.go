package blevesearch

import (
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/blevesearch/bleve"
)

var (
	logger = logging.LoggerForModule()
)

// Indexer is the Bleve implementation of Indexer
type Indexer struct {
	alertIndex      bleve.Index
	deploymentIndex bleve.Index
	imageIndex      bleve.Index
	policyIndex     bleve.Index
}

// NewIndexer creates a new Indexer based on Bleve
func NewIndexer() (*Indexer, error) {
	b := &Indexer{}
	if err := b.initializeIndices(); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Indexer) initializeIndices() error {
	var err error

	b.alertIndex, err = bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		return err
	}

	b.deploymentIndex, err = bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		return err
	}

	b.imageIndex, err = bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		return err
	}

	b.policyIndex, err = bleve.NewMemOnly(bleve.NewIndexMapping())
	return err
}

// Close closes the open indexes
func (b *Indexer) Close() error {
	if err := b.alertIndex.Close(); err != nil {
		return err
	}
	if err := b.deploymentIndex.Close(); err != nil {
		return err
	}
	if err := b.imageIndex.Close(); err != nil {
		return err
	}
	return b.policyIndex.Close()
}
