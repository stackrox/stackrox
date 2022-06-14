package cryptoutils

import (
	"time"

	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/timeutil"
)

// NoncePool is a source and consumer of nonces. It can be used to issue short-lived nonces, and verify that every
// issued nonce is consumed at most once.
type NoncePool interface {
	IssueNonce() (string, error)
	ConsumeNonce(nonce string) bool
}

// NewThreadSafeNoncePool returns a new thread-safe nonce pool.
func NewThreadSafeNoncePool(gen NonceGenerator, ttl time.Duration) NoncePool {
	return WrapNoncePoolThreadSafe(NewThreadUnsafeNoncePool(gen, ttl))
}

// NewThreadUnsafeNoncePool creates and returns a new nonce pool that is NOT safe for concurrent use.
func NewThreadUnsafeNoncePool(gen NonceGenerator, ttl time.Duration) NoncePool {
	return &noncePool{
		generator:    gen,
		issuedNonces: make(map[string]time.Time),
		ttl:          ttl,

		nextExpiry: timeutil.Max,
	}
}

// WrapNoncePoolThreadSafe wraps a nonce pool such that accesses to the returned pool are safe for concurrent use.
// The original pool must not be used afterwards.
func WrapNoncePoolThreadSafe(noncePool NoncePool) NoncePool {
	if _, ok := noncePool.(*threadSafeNoncePoolWrap); ok {
		return noncePool
	}
	return &threadSafeNoncePoolWrap{
		pool: noncePool,
	}
}

type noncePool struct {
	generator    NonceGenerator
	issuedNonces map[string]time.Time
	ttl          time.Duration

	nextExpiry time.Time
}

func (p *noncePool) cleanup() {
	now := time.Now()
	if !now.After(p.nextExpiry) {
		return
	}

	nextExpiry := timeutil.Max
	for nonce, expiry := range p.issuedNonces {
		if expiry.Before(now) {
			delete(p.issuedNonces, nonce)
		} else if expiry.Before(nextExpiry) {
			nextExpiry = expiry
		}
	}
	p.nextExpiry = nextExpiry
}

func (p *noncePool) IssueNonce() (string, error) {
	nonce, err := p.generator.Nonce()
	if err != nil {
		return "", err
	}

	now := time.Now()
	expiry := now.Add(p.ttl)
	p.issuedNonces[nonce] = expiry
	if p.nextExpiry.After(expiry) { // can only happen if nextExpiry = timeutil.Max
		p.nextExpiry = expiry
	} else {
		p.cleanup()
	}

	return nonce, nil
}

func (p *noncePool) ConsumeNonce(nonce string) bool {
	expiry, ok := p.issuedNonces[nonce]
	now := time.Now()
	if ok {
		delete(p.issuedNonces, nonce)
		ok = !expiry.Before(now)
	}
	p.cleanup()

	return ok
}

type threadSafeNoncePoolWrap struct {
	pool  NoncePool
	mutex sync.Mutex
}

func (p *threadSafeNoncePoolWrap) IssueNonce() (string, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.pool.IssueNonce()
}

func (p *threadSafeNoncePoolWrap) ConsumeNonce(nonce string) bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.pool.ConsumeNonce(nonce)
}
