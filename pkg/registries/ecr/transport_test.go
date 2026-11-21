package ecr

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	awsECR "github.com/aws/aws-sdk-go-v2/service/ecr"
	awsECRTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestECRTransport(t *testing.T) {
	transport := awsTransport{}
	// Token is invalid initially.
	assert.False(t, transport.isValidNoLock())

	// Token is valid after the first token exchange.
	expiredAt := time.Now().Add(12 * time.Hour)
	transport.expiresAt = &expiredAt
	assert.True(t, transport.isValidNoLock())

	// Token is valid if more than the expiry delta is left.
	expiredAt = time.Now().Add(360 * time.Second)
	transport.expiresAt = &expiredAt
	assert.True(t, transport.isValidNoLock())

	// Token is invalid if less than the expiry delta is left.
	expiredAt = time.Now().Add(180 * time.Second)
	transport.expiresAt = &expiredAt
	assert.False(t, transport.isValidNoLock())

	// Token is invalid after it expired.
	expiredAt = time.Now()
	transport.expiresAt = &expiredAt
	assert.False(t, transport.isValidNoLock())
}

type fakeECRClient struct {
	blockUntilCtxDone bool
	sleep             time.Duration
	output            *awsECR.GetAuthorizationTokenOutput
	err               error
}

func (f *fakeECRClient) GetAuthorizationToken(ctx context.Context, _ *awsECR.GetAuthorizationTokenInput, _ ...func(*awsECR.Options)) (*awsECR.GetAuthorizationTokenOutput, error) {
	if f.blockUntilCtxDone {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}
	return f.output, f.err
}

func validTokenOutput(t *testing.T) *awsECR.GetAuthorizationTokenOutput {
	t.Helper()
	token := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	expiresAt := time.Now().Add(12 * time.Hour)
	return &awsECR.GetAuthorizationTokenOutput{
		AuthorizationData: []awsECRTypes.AuthorizationData{
			{
				AuthorizationToken: &token,
				ExpiresAt:          &expiresAt,
			},
		},
	}
}

func TestRefreshNoLock(t *testing.T) {
	cases := map[string]struct {
		client         *fakeECRClient
		refreshTimeout time.Duration
		expectErr      bool
		expectTimeout  bool
	}{
		"blocked client is bounded by configured timeout": {
			client:         &fakeECRClient{blockUntilCtxDone: true},
			refreshTimeout: 50 * time.Millisecond,
			expectErr:      true,
			expectTimeout:  true,
		},
		"happy path returns before timeout": {
			client:         &fakeECRClient{output: validTokenOutput(t)},
			refreshTimeout: 50 * time.Millisecond,
			expectErr:      false,
		},
		"slow but within budget still succeeds": {
			client:         &fakeECRClient{sleep: 10 * time.Millisecond, output: validTokenOutput(t)},
			refreshTimeout: 200 * time.Millisecond,
			expectErr:      false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			transport := &awsTransport{
				client:         c.client,
				config:         &docker.Config{},
				refreshTimeout: c.refreshTimeout,
			}

			errCh := make(chan error, 1)
			go func() {
				errCh <- transport.refreshNoLock(context.Background())
			}()

			select {
			case err := <-errCh:
				if c.expectErr {
					require.Error(t, err)
					if c.expectTimeout {
						assert.ErrorIs(t, err, context.DeadlineExceeded)
					}
				} else {
					require.NoError(t, err)
					assert.NotNil(t, transport.expiresAt)
				}
			case <-time.After(time.Second):
				t.Fatal("refreshNoLock did not return within the safety-net timeout")
			}
		})
	}
}
