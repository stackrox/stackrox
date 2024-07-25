package generic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendProtoPayload(t *testing.T) {
	fakeSumoLogicSvc := &fakeSumoLogic{
		tb:              t,
		expectedPayload: fixtures.GetJSONSerializedTestAlert(),
	}
	server := httptest.NewServer(fakeSumoLogicSvc)
	defer server.Close()

	notifierConfig := &storage.Notifier{
		Config: &storage.Notifier_Sumologic{
			Sumologic: &storage.SumoLogic{
				HttpSourceAddress: server.URL,
				SkipTLSVerify:     true,
			},
		},
	}
	sumoLogicNotifier, err := newSumoLogic(notifierConfig)
	require.NoError(t, err)

	err = sumoLogicNotifier.sendProtoPayload(context.Background(), fixtures.GetSerializationTestAlert())
	assert.NoError(t, err)
}

type fakeSumoLogic struct {
	tb              testing.TB
	expectedPayload string
}

func (s *fakeSumoLogic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		s.tb.Error("Bad HTTP method", r.Method)
		return
	}

	body := r.Body
	defer func() { _ = body.Close() }()
	bodyData, err := io.ReadAll(body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.tb.Error("Error reading body", err)
		return
	}

	match := assert.JSONEq(s.tb, s.expectedPayload, string(bodyData))
	if !match {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/text")
	w.WriteHeader(200)
	_, err = w.Write([]byte("ok"))
	assert.NoError(s.tb, err)
}
