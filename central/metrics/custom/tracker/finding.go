package tracker

import "github.com/pkg/errors"

// ErrStopIterator can be set to a finding to signal that yield returned false,
// and so doesn't accept any more findings.
var ErrStopIterator = errors.New("stopped")

type Finding interface {
	GetError() error
	GetIncrement() int
}

type FindingBase struct{}

func (*FindingBase) GetError() error { return nil }

func (*FindingBase) GetIncrement() int { return 1 }

type FindingWithErr struct {
	FindingBase
	err error
}

func (f *FindingWithErr) GetError() error { return f.err }

func (f *FindingWithErr) SetError(err error) { f.err = err }

// Collector yields a finding. Returns ErrStopIterator if yield returns false,
// and otherwise the finding error.
type Collector[F Finding] func(F) error

// NewFindingCollector returns a finding collector.
// The collector function wraps the yield function with an error handling.
//
// Example:
//
//	var finding MyFinding
//	collector := NewFindingCollector(yield)
//
//	finding.SetError(walk(objs, func(obj O) error {
//	   finding.obj = obj
//	   return collector(&finding)
//	}
//	collector.Finally(&finding)
func NewFindingCollector[F Finding](yield func(F) bool) Collector[F] {
	return func(f F) error {
		err := f.GetError()
		if !errors.Is(err, ErrStopIterator) && !yield(f) {
			if err == nil {
				err = ErrStopIterator
			} else {
				err = errors.Wrap(ErrStopIterator, err.Error())
			}
		}
		return err
	}
}

// Finally yields the finding with an error, if it is not ErrStopIterator.
func (c Collector[F]) Finally(f F) {
	if err := f.GetError(); err != nil && !errors.Is(err, ErrStopIterator) {
		_ = c(f)
	}
}
