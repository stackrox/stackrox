package authn

import (
	"context"
	"errors"
	"fmt"

	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"gopkg.in/square/go-jose.v2/jwt"
)

// ExtractorError represents an error that occurs during identity extraction.
type ExtractorError struct {
	// extractorType identifies the type of identity extractor that encountered the error.
	extractorType string

	// msg provides additional information or context about the error.
	msg string

	// err holds the underlying error.
	err error
}

// NewExtractorError creates and returns a new instance of ExtractorError.
//
// Parameters:
//   - extractorType: A string representing the type of identity extractor where the error occurred.
//   - msg: A string providing additional information or context about the error.
//   - err: The underlying error that caused the extraction failure.
//
// Returns:
//   - An initialized pointer to an ExtractorError struct with the given parameters.
func NewExtractorError(extractorType string, msg string, err error) *ExtractorError {
	return &ExtractorError{
		extractorType: extractorType,
		msg:           msg,
		err:           err,
	}
}

var _ error = (*ExtractorError)(nil)

func (e *ExtractorError) Error() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf("%v: cannot extract identity: %v", e.extractorType, e.msg)
}

func (e *ExtractorError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

// LogL logs the details of the ExtractorError, associating it with the provided RequestInfo.
//
// Parameters:
//   - ri: The RequestInfo associated with the ExtractorError.
func (e *ExtractorError) LogL(ri requestinfo.RequestInfo) {
	if e == nil {
		return
	}

	// We are handling some errors differently because
	// they are frequent and expected.
	logF := logging.GetRateLimitedLogger().WarnL
	if errors.Is(e.Unwrap(), jwt.ErrExpired) {
		logF = logging.GetRateLimitedLogger().DebugL
	}

	// We might print nil at the end, like:
	//    "Cannot ... [basic] for hostname example.com: parse error: <nil>"
	// but this is alright in our logs.
	logF(
		ri.Hostname,
		"Cannot extract identity [%v] for hostname %v: %v: %v",
		e.extractorType,
		ri.Hostname,
		e.msg,
		e.err,
	)
}

// ValidateCertChain can be implemented to provide cert chain validation callbacks
type ValidateCertChain interface {
	// ValidateClientCertificate validates the given certificate chain
	ValidateClientCertificate(context.Context, []mtls.CertInfo) error
}

// IdentityExtractor extracts the identity of a user making a request from a request info.
type IdentityExtractor interface {
	IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (Identity, *ExtractorError)
}

type extractorList []IdentityExtractor

func (l extractorList) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (Identity, *ExtractorError) {
	for _, extractor := range l {
		if id, err := extractor.IdentityForRequest(ctx, ri); id != nil || err != nil {
			return id, err
		}
	}
	return nil, nil
}

// CombineExtractors combines the given identity extractors.
func CombineExtractors(extractors ...IdentityExtractor) IdentityExtractor {
	if len(extractors) == 1 {
		return extractors[0]
	}
	return extractorList(extractors)
}
