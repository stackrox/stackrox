package signer

import (
	"crypto/rsa"
	"time"

	"bitbucket.org/stack-rox/apollo/central/apitoken"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"gopkg.in/square/go-jose.v2/jwt"
)

type signerImpl struct {
	keyID   uuid.UUID
	key     *rsa.PrivateKey
	builder jwt.Builder
}

func (s *signerImpl) SignedJWT(subject string) (string, error) {
	now := time.Now()
	claims := jwt.Claims{
		Audience: []string{apitoken.Audience},
		Subject:  subject,
		Issuer:   apitoken.Issuer,
		Expiry:   jwt.NewNumericDate(now.Add(apitoken.DefaultExpiry)),
		IssuedAt: jwt.NewNumericDate(now),
		ID:       uuid.NewV4().String(),
	}
	return s.builder.Claims(claims).CompactSerialize()
}

func (s *signerImpl) Key(id string) (interface{}, bool) {
	if id == s.keyID.String() {
		return s.key.Public(), true
	}
	return nil, false
}
