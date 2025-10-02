package postgres

// ReindexGCOption is a configuration option for the GCManifests method.
type ReindexGCOption func(o *reindexGCOpts)

type reindexGCOpts struct {
	gcThrottle int
}

// WithGCThrottle sets the maximum number of manifests to GC.
// Default: 100
func WithGCThrottle(gcThrottle int) ReindexGCOption {
	return func(o *reindexGCOpts) {
		o.gcThrottle = gcThrottle
	}
}

func makeReindexGCOpts(opts []ReindexGCOption) reindexGCOpts {
	var o reindexGCOpts
	for _, opt := range opts {
		opt(&o)
	}

	if o.gcThrottle == 0 {
		o.gcThrottle = 100
	}

	return o
}
