package csv

type writerOptions struct {
	header       []string
	rowConverter any
	withBOM      bool
	withCRLF     bool
	delimiter    rune
}

// Option modifies the provided CallOptions structure.
type Option func(*writerOptions)

// WithHeader sets the CSV header record.
func WithHeader(header ...string) Option {
	return func(o *writerOptions) {
		o.header = header
	}
}

// WithNoHeader instructs to not write a header.
func WithNoHeader() Option {
	return func(o *writerOptions) {
		o.header = []string{}
	}
}

// WithBOM instructs to prepend the output with UTF-8 BOM sequence.
func WithBOM() Option {
	return func(o *writerOptions) {
		o.withBOM = true
	}
}

// WithCRLF configures the writer to use \r\n line endings.
func WithCRLF() Option {
	return func(o *writerOptions) {
		o.withCRLF = true
	}
}

// WithDelimiter configures the writer to use the provided delimiter instead of
// a comma.
func WithDelimiter(d rune) Option {
	return func(o *writerOptions) {
		o.delimiter = d
	}
}

// WithConverter allows for providing a custom conversion from a record object
// to a slice of strings. If not used, the default reflection based conversion
// will do the job.
func WithConverter[Row any](fn RowConverterFunc[Row]) Option {
	return func(o *writerOptions) {
		o.rowConverter = fn
	}
}
