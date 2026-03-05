package service

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/auth/tokens"
	tokenMocks "github.com/stackrox/rox/pkg/auth/tokens/mocks"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	testPurgeDelay = time.Minute

	testAudience = "test audience"
	fakeAudience = "fake audience"
)

func TestSomething(t *testing.T) {
	assert.True(t, true)
}

func TestGetIssuerFromCache(t *testing.T) {
	t.Run("cache miss", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		issuer, found := mgr.getIssuerFromCache(testAudience, testTokenExpiry)
		assert.False(it, found)
		assert.Nil(it, issuer)
	})
	t.Run("cache hit updates cache expiry time", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mockSrc := tokenMocks.NewMockSource(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		addIssuerToManager(mgr, fakeAudience, mockSrc, mockIssuer, testClockTime)

		issuer, found := mgr.getIssuerFromCache(fakeAudience, testTokenExpiry)
		assert.True(it, found)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, fakeAudience, true, mockIssuer, testTokenExpiry)
	})
}

func TestAddIssuerToCache(t *testing.T) {
	t.Run("error from issuer factory is propagated", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		mockIssuerFactory.EXPECT().
			CreateIssuer(gomock.Any(), gomock.Any()).
			Times(1).
			Return(nil, errDummy)
		issuer, err := mgr.addIssuerToCache(testAudience, testTokenExpiry)
		assert.Nil(it, issuer)
		assert.ErrorIs(it, err, errDummy)
	})
	t.Run("cache miss creates a new issuer and stores it in cache", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		mockIssuer := setExpectIssue(mockCtrl, mockIssuerFactory)

		issuer, err := mgr.addIssuerToCache(testAudience, testTokenExpiry)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
	})
	t.Run("cache hit does not create a new issuer and updates the expiry time", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		addIssuerToManager(mgr, testAudience, nil, mockIssuer, testClockTime)

		issuer, err := mgr.addIssuerToCache(testAudience, testTokenExpiry)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
	})
}

func TestGetIssuer(t *testing.T) {
	t.Run("factory provides the issuer if none is in cache", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		mockIssuer := setExpectIssue(mockCtrl, mockIssuerFactory)

		issuer, err := mgr.getIssuer(testAudience, testTokenExpiry)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
	})
	t.Run("cache serves the issuer if already in cache", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		addIssuerToManager(mgr, testAudience, nil, mockIssuer, testClockTime)

		issuer, err := mgr.getIssuer(testAudience, testTokenExpiry)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
	})
	t.Run("factory provides the issuer, then cache serves later requests", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		mockIssuer := setExpectIssue(mockCtrl, mockIssuerFactory)

		// First call: generate and cache
		issuer, err := mgr.getIssuer(testAudience, testClockTime)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testClockTime)
		// Second call: serve from cache (and push expiry)
		issuer, err = mgr.getIssuer(testAudience, testTokenExpiry)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
		// Third call: serve from cache
		// (cache expiry is after requested one, left unchanged as a consequence)
		issuer, err = mgr.getIssuer(testAudience, testClockTime)
		assert.NoError(it, err)
		assert.Equal(it, mockIssuer, issuer)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
	})
}

func TestPurgeExpired(t *testing.T) {
	t.Run("Purge on an empty cache does not touch anything", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		assert.Empty(it, mgr.cache)

		mgr.purgeExpired(testClockTime)
		assert.Empty(it, mgr.cache)
	})
	t.Run("Purge on a cache with a single non-expired item does not touch anything", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		addIssuerToManager(mgr, testAudience, nil, mockIssuer, testTokenExpiry)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)

		mgr.purgeExpired(testClockTime)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testTokenExpiry)
	})
	t.Run("Purge on a cache with a single expired item removes the expired item", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		addIssuerToManager(mgr, testAudience, nil, mockIssuer, testClockTime.Add(-1*time.Minute))
		assert.Len(it, mgr.cache, 1)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, testClockTime.Add(-1*time.Minute))
		mockIssuerFactory.EXPECT().UnregisterSource(gomock.Any()).Times(1).Return(nil)

		mgr.purgeExpired(testClockTime)
		assert.Empty(it, mgr.cache)
	})
	t.Run("Purge on a cache with one item expired and one not only removes the expired item", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		expired := testClock().Add(-1 * time.Minute)
		notExpired := testClock().Add(1 * time.Minute)
		addIssuerToManager(mgr, fakeAudience, nil, mockIssuer, expired)
		addIssuerToManager(mgr, testAudience, nil, mockIssuer, notExpired)
		assert.Len(it, mgr.cache, 2)
		validateCacheEntry(it, mgr, fakeAudience, true, mockIssuer, expired)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, notExpired)
		mockIssuerFactory.EXPECT().UnregisterSource(gomock.Any()).Times(1).Return(nil)

		mgr.purgeExpired(testClockTime)
		assert.Len(it, mgr.cache, 1)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, notExpired)
	})
	t.Run("Purge on a cache with multiple expired items removes the expired items", func(it *testing.T) {
		mockCtrl := gomock.NewController(it)
		mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
		mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
		mgr := newIssuerManager(mockIssuerFactory, testPurgeDelay)
		old := testClock().Add(-1 * time.Minute)
		veryOld := testClock().Add(-5 * time.Minute)
		addIssuerToManager(mgr, fakeAudience, nil, mockIssuer, old)
		addIssuerToManager(mgr, testAudience, nil, mockIssuer, veryOld)
		assert.Len(it, mgr.cache, 2)
		validateCacheEntry(it, mgr, fakeAudience, true, mockIssuer, old)
		validateCacheEntry(it, mgr, testAudience, true, mockIssuer, veryOld)
		mockIssuerFactory.EXPECT().UnregisterSource(gomock.Any()).Times(2).Return(nil)

		mgr.purgeExpired(testClockTime)
		assert.Empty(it, mgr.cache)
	})
}

func TestPurgeWorkflow(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockIssuerFactory := tokenMocks.NewMockIssuerFactory(mockCtrl)
	mockSource := tokenMocks.NewMockSource(mockCtrl)
	mockSource.EXPECT().ID().AnyTimes().Return(testAudience)
	mockIssuer := tokenMocks.NewMockIssuer(mockCtrl)
	mgr := newIssuerManager(mockIssuerFactory, 10*time.Millisecond)

	// blank purge run (timer not set)
	mgr.purge()

	// reset
	mgr = newIssuerManager(mockIssuerFactory, 10*time.Millisecond)

	wg := sync.WaitGroup{}
	wg.Add(1)
	addIssuerToManager(mgr, testAudience, mockSource, mockIssuer, testClock())
	mockIssuerFactory.EXPECT().UnregisterSource(mockSource).Times(1).DoAndReturn(
		func(_ tokens.Source) error {
			wg.Done()
			return nil
		},
	)

	mgr.Start()
	wg.Wait()
	assert.Empty(t, mgr.cache)

	mgr.Stop()
}

func addIssuerToManager(
	mgr *issuerManager,
	audience string,
	src tokens.Source,
	issuer tokens.Issuer,
	expiresAt time.Time,
) {
	mgr.cache[audience] = &issuerWrapper{
		source:    src,
		issuer:    issuer,
		expiresAt: expiresAt,
	}
}

func setExpectIssue(mockCtrl *gomock.Controller, factory *tokenMocks.MockIssuerFactory) tokens.Issuer {
	issuer := tokenMocks.NewMockIssuer(mockCtrl)
	factory.EXPECT().
		CreateIssuer(gomock.Any(), gomock.Any()).
		Times(1).
		Return(issuer, nil)
	return issuer
}

func validateCacheEntry(
	t *testing.T,
	mgr *issuerManager,
	key string,
	expectedFound bool,
	issuer tokens.Issuer,
	expiresAt time.Time,
) {
	t.Helper()
	iw, found := mgr.cache[key]
	if expectedFound {
		assert.True(t, found)
		assert.NotNil(t, iw)
		assert.Equal(t, issuer, iw.issuer)
		assert.Equal(t, expiresAt, iw.expiresAt)
	} else {
		assert.False(t, found)
		assert.Nil(t, iw)
	}
}
