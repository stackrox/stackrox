package idputil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateStore_IssueAndRedeem(t *testing.T) {
	store := NewStateStore(5 * time.Minute)

	cases := map[string]struct {
		providerID  string
		clientState string
	}{
		"simple": {
			providerID:  "provider-123",
			clientState: "some-state",
		},
		"empty client state": {
			providerID:  "provider-123",
			clientState: "",
		},
		"roxctl authorize with URL": {
			providerID:  "provider-123",
			clientState: AuthorizeRoxctlClientState + "#http://localhost:12345/callback?foo=bar",
		},
		"test mode prefix": {
			providerID:  "provider-123",
			clientState: TestLoginClientState + "#original-state",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			nonce, err := store.Issue(tc.providerID, tc.clientState)
			require.NoError(t, err)
			assert.NotEmpty(t, nonce)

			gotProvider, gotClient, err := store.Redeem(nonce)
			require.NoError(t, err)
			assert.Equal(t, tc.providerID, gotProvider)
			assert.Equal(t, tc.clientState, gotClient)
		})
	}
}

func TestStateStore_RedeemConsumesSingleUse(t *testing.T) {
	store := NewStateStore(5 * time.Minute)

	nonce, err := store.Issue("provider", "state")
	require.NoError(t, err)

	_, _, err = store.Redeem(nonce)
	require.NoError(t, err)

	_, _, err = store.Redeem(nonce)
	assert.ErrorIs(t, err, errNonceUnknown)
}

func TestStateStore_LookupIsNonConsuming(t *testing.T) {
	store := NewStateStore(5 * time.Minute)

	nonce, err := store.Issue("provider", "state")
	require.NoError(t, err)

	for range 3 {
		p, c, err := store.Lookup(nonce)
		require.NoError(t, err)
		assert.Equal(t, "provider", p)
		assert.Equal(t, "state", c)
	}

	p, c, err := store.Redeem(nonce)
	require.NoError(t, err)
	assert.Equal(t, "provider", p)
	assert.Equal(t, "state", c)
}

func TestStateStore_ExpiredNonceFails(t *testing.T) {
	store := NewStateStore(1 * time.Millisecond)

	nonce, err := store.Issue("provider", "state")
	require.NoError(t, err)

	time.Sleep(5 * time.Millisecond)

	_, _, err = store.Redeem(nonce)
	assert.ErrorIs(t, err, errNonceUnknown)

	_, _, err = store.Lookup(nonce)
	assert.ErrorIs(t, err, errNonceUnknown)
}

func TestStateStore_UnknownNonceFails(t *testing.T) {
	store := NewStateStore(5 * time.Minute)

	_, _, err := store.Redeem("nonexistent-nonce")
	assert.ErrorIs(t, err, errNonceUnknown)

	_, _, err = store.Lookup("nonexistent-nonce")
	assert.ErrorIs(t, err, errNonceUnknown)
}

func TestStateStore_IndependentNonces(t *testing.T) {
	store := NewStateStore(5 * time.Minute)

	nonce1, err := store.Issue("provider-1", "state-1")
	require.NoError(t, err)

	nonce2, err := store.Issue("provider-2", "state-2")
	require.NoError(t, err)

	assert.NotEqual(t, nonce1, nonce2)

	p, c, err := store.Redeem(nonce1)
	require.NoError(t, err)
	assert.Equal(t, "provider-1", p)
	assert.Equal(t, "state-1", c)

	p, c, err = store.Redeem(nonce2)
	require.NoError(t, err)
	assert.Equal(t, "provider-2", p)
	assert.Equal(t, "state-2", c)
}

func TestIssueStateNonce_PackageLevel(t *testing.T) {
	nonce, err := IssueStateNonce("provider-id", "client-state")
	require.NoError(t, err)

	p, c, err := RedeemStateNonce(nonce)
	require.NoError(t, err)
	assert.Equal(t, "provider-id", p)
	assert.Equal(t, "client-state", c)
}

func TestLookupStateNonce_PackageLevel(t *testing.T) {
	nonce, err := IssueStateNonce("provider-id", "client-state")
	require.NoError(t, err)

	p, c, err := LookupStateNonce(nonce)
	require.NoError(t, err)
	assert.Equal(t, "provider-id", p)
	assert.Equal(t, "client-state", c)

	// Cleanup: redeem so it doesn't leak.
	_, _, _ = RedeemStateNonce(nonce)
}
