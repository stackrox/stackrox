package batcher

// Batcher takes in a total size and a batch size and returns the indices for the next batch
type Batcher struct {
	curr  int
	total int
	batch int
}

// New returns new batcher
func New(totalSize, batchSize int) *Batcher {
	return &Batcher{
		total: totalSize,
		batch: batchSize,
	}
}

// Next returns the next [start,end) indices and if the next batch is valid
func (b *Batcher) Next() (start int, end int, valid bool) {
	if b.curr >= b.total {
		valid = false
		return
	}
	valid = true
	start = b.curr
	if end = b.curr + b.batch; end > b.total {
		end = b.total
	}

	b.curr += b.batch
	return
}
