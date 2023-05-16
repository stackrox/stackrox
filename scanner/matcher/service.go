package matcher

// TODO: Make this into a gRPC service.
type Service struct {
	matcher *Matcher
}

func NewService(matcher *Matcher) (*Service, error) {
	return &Service{
		matcher: matcher,
	}, nil
}
