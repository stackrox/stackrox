package certwatch

// CertWatchConfig configures how certificates are watched.
type CertWatchConfig struct {
	Verify bool
}

// GetVerify returns the verification state.
func (c *CertWatchConfig) GetVerify() bool {
	if c == nil {
		return true
	}
	return c.Verify
}

// CertWatchOption is a functor that applies the certwatch config option.
type CertWatchOption func(opt *CertWatchConfig) *CertWatchConfig

func applyOptions(options ...CertWatchOption) *CertWatchConfig {
	cfg := &CertWatchConfig{Verify: true}
	for _, opt := range options {
		cfg = opt(cfg)
	}
	return cfg
}

// WithVerify enables/disables the verification of watched certificates.
func WithVerify(verify bool) CertWatchOption {
	return func(cfg *CertWatchConfig) *CertWatchConfig {
		if cfg == nil {
			cfg = &CertWatchConfig{}
		}
		cfg.Verify = verify
		return cfg
	}
}
