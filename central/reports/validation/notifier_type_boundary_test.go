package validation

import (
	"testing"

	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	apiV2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateEmailConfigRejectsNonEmailNotifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	notifierDS := notifierDSMocks.NewMockDataStore(ctrl)
	notifierDS.EXPECT().GetScrubbedNotifier(gomock.Any(), "notifier").Return(&storage.Notifier{Type: "slack"}, true, nil)

	err := (&Validator{notifierDatastore: notifierDS}).validateEmailConfig(&apiV2.EmailNotifierConfiguration{
		NotifierId: "notifier", MailingLists: []string{"reader@example.com"},
	})
	require.ErrorIs(t, err, errox.InvalidArgs)
}
