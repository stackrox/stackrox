package env

type durationSettingOpts struct {
	zeroAllowed bool
}

// DurationSettingOption represents an option which may be specified
// for a DurationSetting environment variable.
type DurationSettingOption interface {
	apply(opts *durationSettingOpts)
}

type durationSettingOptsFunc func(opts *durationSettingOpts)

func (f durationSettingOptsFunc) apply(opts *durationSettingOpts) {
	f(opts)
}

// WithDurationZeroAllowed specifies the DurationSetting may have a value of 0.
func WithDurationZeroAllowed() DurationSettingOption {
	return durationSettingOptsFunc(func(opts *durationSettingOpts) {
		opts.zeroAllowed = true
	})
}
