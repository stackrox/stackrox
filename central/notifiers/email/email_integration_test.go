//go:build integration
// +build integration

package email

import (
	"context"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	mitreMocks "github.com/stackrox/rox/central/mitre/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	server   = "smtp.mailgun.org"
	user     = "postmaster@sandboxd6576ea8be3c477989eba2c14735d2e6.mailgun.org"
	password = "ec221fbe09156dfef5bd48afb71d277c"

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
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	nsStore.EXPECT().SearchNamespaces(gomock.Any(), gomock.Any()).Return([]*storage.NamespaceMetadata{}, nil).AnyTimes()
	mitreStore := mitreMocks.NewMockMitreAttackReadOnlyDataStore(mockCtrl)
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

	e, err := newEmail(notifier, nsStore, mitreStore)
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
