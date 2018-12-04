package scanners

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeScanner struct {
	global bool
}

func (*fakeScanner) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	panic("implement me")
}

func (f *fakeScanner) Global() bool {
	return f.global
}

func (*fakeScanner) Match(image *v1.Image) bool {
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
		fmt.Fprintf(w, "{}")
	}))
	defer server.Close()

	registryFactory := registries.NewFactory()
	registrySet := registries.NewSet(registryFactory)

	scannerFactory := NewFactory(registrySet)
	scannerSet := NewSet(scannerFactory)

	clairifyIntegration := &v1.ImageIntegration{
		Id:   "clairify",
		Type: "clairify",
		IntegrationConfig: &v1.ImageIntegration_Clairify{
			Clairify: &v1.ClairifyConfig{
				Endpoint: server.URL,
			},
		},
	}
	dtrIntegration := &v1.ImageIntegration{
		Id:   "dtr",
		Type: "dtr",
		IntegrationConfig: &v1.ImageIntegration_Dtr{
			Dtr: &v1.DTRConfig{
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

	goodIntegration := &v1.ImageIntegration{Id: "GOOD"}
	mockFactory.EXPECT().CreateScanner(goodIntegration).Return(&fakeScanner{global: true}, nil).Times(2)

	badIntegration := &v1.ImageIntegration{Id: "BAD"}
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
