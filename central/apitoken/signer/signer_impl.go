package signer

import (
	"crypto/rsa"
	"time"

	"github.com/stackrox/rox/central/apitoken"
	"github.com/stackrox/rox/pkg/uuid"
	"gopkg.in/square/go-jose.v2/jwt"
)

type signerImpl struct {
	keyID   uuid.UUID
	key     *rsa.PrivateKey
	builder jwt.Builder
}

func (s *signerImpl) SignedJWT(subject string) (token, id string, issuedAt, expiration time.Time, err error) {
	id = uuid.NewV4().String()
	issuedAt = time.Now()
	expiration = issuedAt.Add(apitoken.DefaultExpiry)

	claims := jwt.Claims{
		Audience: []string{apitoken.Audience},
		Subject:  subject,
		Issuer:   apitoken.Issuer,
		Expiry:   jwt.NewNumericDate(expiration),
		IssuedAt: jwt.NewNumericDate(issuedAt),
		ID:       id,
	}

	token, err = s.builder.Claims(claims).CompactSerialize()
	return
}

func (s *signerImpl) Key(id string) (interface{}, bool) {
	if id == s.keyID.String() {
		return s.key.Public(), true
	}
	return nil, false
}
