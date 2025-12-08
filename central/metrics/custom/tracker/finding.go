package tracker

import "github.com/pkg/errors"

// ErrStopIterator can be set to a finding to signal that yield returned false,
// and so doesn't accept any more findings.
var ErrStopIterator = errors.New("stopped")

type Finding any

// WithIncrement is an optional interface for a finding, which allows for
// counting several elements per single finding.
type WithIncrement interface{ GetIncrement() int }

// Collector yields a finding. Returns ErrStopIterator if yield returns false,
// and otherwise the finding error.
type Collector[F Finding] func(F, error) bool

func NewFindingCollector[F Finding](yield func(F, error) bool) Collector[F] {
	return Collector[F](yield)
}

func (c Collector[F]) Yield(f F) error {
	if !c(f, nil) {
		return ErrStopIterator
	}
	return nil
}

func (c Collector[F]) Error(err error) {
	if err != nil {
		var f F
		_ = c(f, err)
	}
}

// Finally yields the finding with an error, if it is not ErrStopIterator.
func (c Collector[F]) Finally(err error) {
	if err != nil && !errors.Is(err, ErrStopIterator) {
		c.Error(err)
	}
}
