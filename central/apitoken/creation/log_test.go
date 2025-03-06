package creation

import (
	"bytes"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_loggingMessage(t *testing.T) {
	ap, err := authproviders.NewProvider(
		authproviders.WithID("1234-5678"),
		authproviders.WithName("provider-name"),
		authproviders.WithType("test-provider"),
	)
	require.NoError(t, err)
	ui := &storage.UserInfo{
		Username:     "username",
		FriendlyName: "friendly name",
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockIdentity := mocks.NewMockIdentity(mockCtrl)
	mockIdentity.EXPECT().User().Times(2).Return(ui)
	mockIdentity.EXPECT().UID().Times(2).Return("0000-0000")
	mockIdentity.EXPECT().ExternalAuthProvider().Times(1).Return(ap)

	buf := make([]byte, 0, 1024)
	w := bytes.NewBuffer(buf)
	log := zap.New(
		zapcore.NewCore(zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			MessageKey: "msg",
			EncodeTime: func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {},
		}),
			zapcore.AddSync(w), zapcore.InfoLevel)).Sugar()

	md := &storage.TokenMetadata{
		Id:    "token-id",
		Name:  "test",
		Roles: []string{"Admin", "Test"},
	}

	LogTokenCreation(log, mockIdentity, md)
	assert.Equal(t, `An API token has been issued	`+
		`{"err_code": "token-created", "api_token_name": "test", "api_token_id": "token-id", `+
		`"roles": ["Admin", "Test"], "user": "username", "user_id": "0000-0000", `+
		`"user_auth_provider": "test-provider \"provider-name\" 1234-5678"}`+"\n",
		w.String())

	w.Reset()
	mockIdentity.EXPECT().ExternalAuthProvider().Times(1).Return(nil)
	LogTokenCreation(log, mockIdentity, md)
	assert.Equal(t, `An API token has been issued	`+
		`{"err_code": "token-created", "api_token_name": "test", "api_token_id": "token-id", `+
		`"roles": ["Admin", "Test"], "user": "username", "user_id": "0000-0000"}`+"\n",
		w.String())
}
