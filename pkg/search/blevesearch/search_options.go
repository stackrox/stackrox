package blevesearch

type opts struct {
	hook HookForCategory
}

// SearchOption is an option that can be used to customize the execution of a Bleve search.
type SearchOption func(opts *opts) error

// WithHook runs the search with the given option.
func WithHook(hook HookForCategory) SearchOption {
	return func(opts *opts) error {
		opts.hook = hook
		return nil
	}
}
