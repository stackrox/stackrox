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

	ticker  *time.Ticker
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
	m.ticker = time.NewTicker(m.purgeInterval)
	go m.purge()
}

func (m *issuerManager) purge() {
	defer m.stopper.Flow().ReportStopped()
	if m.ticker == nil {
		return
	}
	for {
		select {
		case now := <-m.ticker.C:
			m.purgeExpired(now)
		case <-m.stopper.Flow().StopRequested():
			m.ticker.Stop()
			m.ticker = nil
			return
		}
	}
}

func (m *issuerManager) Stop() {
	m.stopper.Client().Stop()
	_ = m.stopper.Client().Stopped().Wait()
}

func (m *issuerManager) getIssuer(audience string, expiresAt time.Time) (tokens.Issuer, error) {
	issuer, found := m.getIssuerFromCache(audience, expiresAt)
	if found {
		return issuer, nil
	}
	return m.addIssuerToCache(audience, expiresAt)
}

func (m *issuerManager) getIssuerFromCache(audience string, expiresAt time.Time) (tokens.Issuer, bool) {
	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()
	iw, found := m.cache[audience]
	if !found {
		return nil, false
	}
	if expiresAt.After(iw.expiresAt) {
		iw.expiresAt = expiresAt
	}
	return iw.issuer, true
}

func (m *issuerManager) addIssuerToCache(audience string, expiresAt time.Time) (tokens.Issuer, error) {
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
	audiencesToRemove := make([]string, 0, len(m.cache))
	for audience, wrapper := range m.cache {
		if wrapper.expiresAt.Before(now) {
			audiencesToRemove = append(audiencesToRemove, audience)
		}
	}
	for _, audience := range audiencesToRemove {
		wrapper := m.cache[audience]
		if wrapper != nil {
			err := m.factory.UnregisterSource(wrapper.source)
			if err != nil {
				log.Errorf("Failed to unregister source for audience %s: %v", audience, err)
			}
			delete(m.cache, audience)
		}
	}
}
