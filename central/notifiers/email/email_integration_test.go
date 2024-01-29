//go:build integration

package email

import (
	"context"
	"os"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cryptoutils/cryptocodec"
	"github.com/stackrox/rox/pkg/fixtures"
	mitreMocks "github.com/stackrox/rox/pkg/mitre/datastore/mocks"
	notifierMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	server   = "smtp.mailgun.org"
	user     = "postmaster@sandboxd6576ea8be3c477989eba2c14735d2e6.mailgun.org"
	password = "ec221fbe09156dfef5bd48afb71d277c" //#nosec G101

	recipientTestEnv = "EMAIL_RECIPIENT"
)

func skip(t *testing.T) string {
	recipient := os.Getenv(recipientTestEnv)
	if recipient == "" {
		t.Skipf("Skipping email integration test because %v is not defined", recipientTestEnv)
	}
	return recipient
}

func getEmail(t *testing.T) (*email, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	metadataGetter := notifierMocks.NewMockMetadataGetter(mockCtrl)
	metadataGetter.EXPECT().GetAnnotationValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("").AnyTimes()
	mitreStore := mitreMocks.NewMockAttackReadOnlyDataStore(mockCtrl)
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()

	recipient := skip(t)

	notifier := &storage.Notifier{
		UiEndpoint: "http://google.com",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Server:     server,
				Sender:     user,
				Username:   recipient,
				Password:   password,
				DisableTLS: true,
			},
		},
	}

	e, err := newEmail(notifier, metadataGetter, mitreStore, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)
	return e, mockCtrl
}

func getUnauthEmail(t *testing.T) (*email, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	metadataGetter := notifierMocks.NewMockMetadataGetter(mockCtrl)
	metadataGetter.EXPECT().GetAnnotationValue(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("").AnyTimes()
	mitreStore := mitreMocks.NewMockAttackReadOnlyDataStore(mockCtrl)
	mitreStore.EXPECT().Get(gomock.Any()).Return(&storage.MitreAttackVector{}, nil).AnyTimes()

	notifier := &storage.Notifier{
		UiEndpoint: "http://google.com",
		Config: &storage.Notifier_Email{
			Email: &storage.Email{
				Server:                   server,
				Sender:                   user,
				AllowUnauthenticatedSmtp: true,
				DisableTLS:               true,
			},
		},
	}

	e, err := newEmail(notifier, metadataGetter, mitreStore, cryptocodec.Singleton(), "stackrox")
	require.NoError(t, err)
	return e, mockCtrl
}

func TestEmailAlertNotify(t *testing.T) {
	e, mockCtrl := getEmail(t)
	defer mockCtrl.Finish()

	assert.NoError(t, e.AlertNotify(context.Background(), fixtures.GetAlert()))
}

func TestEmailNetworkPolicyYAMLNotify(t *testing.T) {
	e, mockCtrl := getEmail(t)
	defer mockCtrl.Finish()

	assert.NoError(t, e.NetworkPolicyYAMLNotify(context.Background(), fixtures.GetYAML(), "test-cluster"))
}

func TestEmailTest(t *testing.T) {
	e, mockCtrl := getEmail(t)
	defer mockCtrl.Finish()

	assert.NoError(t, e.Test(context.Background()))
}

func TestUnauthEmail(t *testing.T) {
	t.Skip("Skipping till ROX-8113 is fixed")
	e, mockCtrl := getUnauthEmail(t)
	defer mockCtrl.Finish()

	assert.NoError(t, e.Test(context.Background()))
}
