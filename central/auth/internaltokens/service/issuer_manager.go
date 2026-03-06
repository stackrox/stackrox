package service

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders/tokenbased"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type issuerWrapper struct {
	source    tokens.Source
	issuer    tokens.Issuer
	expiresAt time.Time
}

type issuerManager struct {
	factory tokens.IssuerFactory

	purgeInterval time.Duration

	stopper concurrency.Stopper

	cacheMutex sync.Mutex
	cache      map[string]*issuerWrapper
}

func newIssuerManager(factory tokens.IssuerFactory, purgeInterval time.Duration) *issuerManager {
	return &issuerManager{
		factory:       factory,
		purgeInterval: purgeInterval,
		stopper:       concurrency.NewStopper(),
		cache:         make(map[string]*issuerWrapper),
	}
}

func (m *issuerManager) Start() {
	ticker := time.NewTicker(m.purgeInterval)
	go m.purge(ticker)
}

func (m *issuerManager) purge(ticker *time.Ticker) {
	defer m.stopper.Flow().ReportStopped()
	if ticker == nil {
		return
	}
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			m.purgeExpired(now)
		case <-m.stopper.Flow().StopRequested():
			return
		}
	}
}

func (m *issuerManager) Stop() {
	m.stopper.Client().Stop()
	_ = m.stopper.Client().Stopped().Wait()
}

func (m *issuerManager) getIssuer(audience string, expiresAt time.Time) (tokens.Issuer, error) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	// first check the entry was not created in cache between initial cache lookup
	// and cache creation request.
	iw, found := m.cache[audience]
	if found {
		if expiresAt.After(iw.expiresAt) {
			iw.expiresAt = expiresAt
		}
		return iw.issuer, nil
	}
	// if still missing, go for the creation.
	name := fmt.Sprintf("internal token source for %s", audience)
	issuerSource := tokenbased.NewTokenAuthProvider(
		audience,
		name,
		internalToken,
		tokenbased.WithRevocationLayer(tokens.NewRevocationLayer()),
	)
	issuer, err := m.factory.CreateIssuer(issuerSource)
	if err != nil {
		return nil, errors.Wrapf(err, "creating issuer for audience %s", audience)
	}
	wrapper := &issuerWrapper{
		source:    issuerSource,
		issuer:    issuer,
		expiresAt: expiresAt,
	}
	m.cache[audience] = wrapper
	return issuer, nil
}

func (m *issuerManager) purgeExpired(now time.Time) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	for audience, wrapper := range m.cache {
		if wrapper == nil {
			delete(m.cache, audience)
			continue
		}
		if wrapper.expiresAt.Before(now) {
			err := m.factory.UnregisterSource(wrapper.source)
			if err != nil {
				log.Errorf("Failed to unregister source for audience %s: %v", audience, err)
			}
			delete(m.cache, audience)
		}
	}
}
