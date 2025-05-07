package mtls

import "time"

type verificationOptions struct {
	currentTime time.Time
}

func (o *verificationOptions) apply(opts []VerifyCertOption) {
	for _, opt := range opts {
		opt(o)
	}
}

// VerifyCertOption is an additional option for certificate verification.
type VerifyCertOption func(o *verificationOptions)

// WithCurrentTime replaces time.Now() with a custom time when performing certificate verification
func WithCurrentTime(currentTime time.Time) VerifyCertOption {
	return func(o *verificationOptions) {
		o.currentTime = currentTime
	}
}
