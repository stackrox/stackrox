package scanners

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeScanner struct {
	global bool
}

func (*fakeScanner) GetLastScan(image *storage.Image) (*storage.ImageScan, error) {
	panic("implement me")
}

func (f *fakeScanner) Global() bool {
	return f.global
}

func (*fakeScanner) Match(image *storage.Image) bool {
	panic("implement me")
}

func (*fakeScanner) Test() error {
	panic("implement me")
}

func (*fakeScanner) Type() string {
	return "type"
}

func TestSetOrdering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "{}")
		assert.NoError(t, err)
	}))
	defer server.Close()

	registryFactory := registries.NewFactory()
	registrySet := registries.NewSet(registryFactory)

	scannerFactory := NewFactory(registrySet)
	scannerSet := NewSet(scannerFactory)

	clairifyIntegration := &storage.ImageIntegration{
		Id:   "clairify",
		Type: "clairify",
		IntegrationConfig: &storage.ImageIntegration_Clairify{
			Clairify: &storage.ClairifyConfig{
				Endpoint: server.URL,
			},
		},
	}
	dtrIntegration := &storage.ImageIntegration{
		Id:   "dtr",
		Type: "dtr",
		IntegrationConfig: &storage.ImageIntegration_Dtr{
			Dtr: &storage.DTRConfig{
				Username: "user",
				Password: "password",
				Endpoint: server.URL,
			},
		},
	}
	require.NoError(t, scannerSet.UpdateImageIntegration(clairifyIntegration))
	require.NoError(t, scannerSet.UpdateImageIntegration(dtrIntegration))
	for i := 0; i < 10000; i++ {
		scanners := scannerSet.GetAll()
		assert.Equal(t, "dtr", scanners[0].Type())
		assert.Equal(t, "clairify", scanners[1].Type())
	}
}

func TestSet(t *testing.T) {
	const errText = "I WON'T LET YOU"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFactory := mocks.NewMockFactory(ctrl)
	s := NewSet(mockFactory)

	goodIntegration := &storage.ImageIntegration{Id: "GOOD"}
	mockFactory.EXPECT().CreateScanner(goodIntegration).Return(&fakeScanner{global: true}, nil).Times(2)

	badIntegration := &storage.ImageIntegration{Id: "BAD"}
	var nilFS *fakeScanner
	mockFactory.EXPECT().CreateScanner(badIntegration).Return(nilFS, errors.New(errText))

	err := s.UpdateImageIntegration(goodIntegration)
	require.Nil(t, err)

	err = s.UpdateImageIntegration(badIntegration)
	require.NotNil(t, err)
	assert.Equal(t, errText, err.Error())

	all := s.GetAll()
	assert.Len(t, all, 1)
	assert.True(t, all[0].Global())

	s.Clear()
	assert.Len(t, s.GetAll(), 0)

	err = s.UpdateImageIntegration(goodIntegration)
	require.Nil(t, err)
	all = s.GetAll()
	assert.Len(t, all, 1)
	assert.True(t, all[0].Global())

	err = s.RemoveImageIntegration(goodIntegration.GetId())
	require.Nil(t, err)
	assert.Len(t, s.GetAll(), 0)

}
