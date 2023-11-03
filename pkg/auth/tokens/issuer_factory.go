package tokens

import (
	"encoding/json"
	"time"

	"github.com/stackrox/rox/pkg/uuid"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// IssuerFactory allows creating issuers from token sources. The signing key is typically tied to the factory.
//
//go:generate mockgen-wrapper
type IssuerFactory interface {
	// CreateIssuer creates an issuer for the given source. This must only be invoked once per source (ID).
	CreateIssuer(source Source, options ...Option) (Issuer, error)
	UnregisterSource(source Source) error
}

func newIssuerFactory(id string, signer jose.Signer, sources *sourceStore, globalOptions ...Option) IssuerFactory {
	return &issuerFactory{
		id:      id,
		sources: sources,
		builder: jwt.Signed(signer),
		options: globalOptions,
	}
}

type issuerFactory struct {
	id      string
	sources *sourceStore
	builder jwt.Builder
	options []Option
}

func (f *issuerFactory) CreateIssuer(source Source, options ...Option) (Issuer, error) {
	if err := f.sources.Register(source); err != nil {
		return nil, err
	}

	allOptions := make([]Option, len(options)+len(f.options))
	copy(allOptions[:len(options)], options)
	copy(allOptions[len(options):], f.options)

	return &issuerForSource{
		source:  source,
		factory: f,
		options: allOptions,
	}, nil
}

func (f *issuerFactory) createClaims(sourceID string, roxClaims RoxClaims) *Claims {
	return &Claims{
		Claims: jwt.Claims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer:   f.id,
			Audience: jwt.Audience{sourceID},
			ID:       uuid.NewV4().String(),
		},
		RoxClaims: roxClaims,
	}
}

func (f *issuerFactory) encode(claims *Claims) (string, error) {
	return f.builder.Claims(&claims.Claims).Claims(&claims.RoxClaims).Claims(translateExtra(claims.Extra)).CompactSerialize()
}

// translateExtra converts a map[string]json.RawMessage to a map[string]interface{} expected by go-jose.
func translateExtra(extra map[string]json.RawMessage) map[string]interface{} {
	if extra == nil {
		return nil
	}
	result := make(map[string]interface{}, len(extra))
	for k, v := range extra {
		result[k] = v
	}
	return result
}

func (f *issuerFactory) UnregisterSource(source Source) error {
	return f.sources.Unregister(source)
}
