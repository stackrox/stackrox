package tokens

// SourceLayer is a single component in a Source, implementing claim validation logic.
type SourceLayer interface {
	Validate(claims *Claims) error
}

// Source is a source for tokens. It has a unique ID which is embedded in the token, as well as validation logic.
type Source interface {
	SourceLayer

	ID() string
}
