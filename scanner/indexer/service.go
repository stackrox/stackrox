package indexer

// TODO: Make this into a gRPC service.
type Service struct {
	indexer *Indexer
}

func NewService(indexer *Indexer) (*Service, error) {
	return &Service{
		indexer: indexer,
	}, nil
}
