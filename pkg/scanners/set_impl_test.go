package scanners

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

var _ types.Scanner = (*fakeScanner)(nil)

type fakeScanner struct {
	typ string
}

func (*fakeScanner) GetScan(*storage.Image) (*storage.ImageScan, error) {
	panic("implement me")
}

func (*fakeScanner) Match(*storage.ImageName) bool {
	panic("implement me")
}

func (*fakeScanner) Test() error {
	panic("implement me")
}

func (*fakeScanner) Name() string {
	panic("implement me")
}

func (f *fakeScanner) Type() string {
	if f.typ != "" {
		return f.typ
	}
	return "type"
}

func (*fakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(10)
}

func (*fakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

var _ types.ImageScannerWithDataSource = (*fakeImageScanner)(nil)

type fakeImageScanner struct {
	scanner types.Scanner
}

func newFakeImageScanner(scanner types.Scanner) types.ImageScannerWithDataSource {
	return &fakeImageScanner{scanner: scanner}
}

func (f *fakeImageScanner) GetScanner() types.Scanner {
	return f.scanner
}

func (*fakeImageScanner) DataSource() *storage.DataSource {
	return nil
}

func TestSetOrdering(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprint(w, "{}")
		assert.NoError(t, err)
	}))
	defer server.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	scannerFactory := mocks.NewMockFactory(ctrl)
	scannerSet := NewSet(scannerFactory)

	scannerFactory.EXPECT().CreateScanner(testutils.PredMatcher("clairify", func(integration *storage.ImageIntegration) bool {
		return integration.GetType() == types.Clairify
	})).Return(newFakeImageScanner(&fakeScanner{typ: types.Clairify}), nil)

	clairifyIntegration := &storage.ImageIntegration{
		Id:   "clairify",
		Type: types.Clairify,
		IntegrationConfig: &storage.ImageIntegration_Clairify{
			Clairify: &storage.ClairifyConfig{
				Endpoint: server.URL,
			},
		},
	}

	scannerFactory.EXPECT().CreateScanner(testutils.PredMatcher("ecr", func(integration *storage.ImageIntegration) bool {
		return integration.GetType() == "ecr"
	})).Return(newFakeImageScanner(&fakeScanner{typ: "ecr"}), nil)

	ecrIntegration := &storage.ImageIntegration{
		Id:   "ecr",
		Type: "ecr",
		IntegrationConfig: &storage.ImageIntegration_Ecr{
			Ecr: &storage.ECRConfig{
				AccessKeyId:     "user",
				SecretAccessKey: "password",
				Endpoint:        server.URL,
			},
		},
	}

	scannerFactory.EXPECT().CreateScanner(testutils.PredMatcher("scannerv4", func(integration *storage.ImageIntegration) bool {
		return integration.GetType() == types.ScannerV4
	})).Return(newFakeImageScanner(&fakeScanner{typ: types.ScannerV4}), nil)

	scannerV4Integration := &storage.ImageIntegration{
		Id:   "scannerv4",
		Type: types.ScannerV4,
		IntegrationConfig: &storage.ImageIntegration_ScannerV4{
			ScannerV4: &storage.ScannerV4Config{},
		},
	}

	require.NoError(t, scannerSet.UpdateImageIntegration(clairifyIntegration))
	require.NoError(t, scannerSet.UpdateImageIntegration(ecrIntegration))
	require.NoError(t, scannerSet.UpdateImageIntegration(scannerV4Integration))
	for i := 0; i < 10000; i++ {
		scanners := scannerSet.GetAll()
		assert.Equal(t, "ecr", scanners[0].GetScanner().Type())
		assert.Equal(t, types.ScannerV4, scanners[1].GetScanner().Type())
		assert.Equal(t, types.Clairify, scanners[2].GetScanner().Type())
	}
}

func TestSet(t *testing.T) {
	const errText = "I WON'T LET YOU"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockFactory := mocks.NewMockFactory(ctrl)
	s := NewSet(mockFactory)

	assert.True(t, s.IsEmpty())

	goodIntegration := &storage.ImageIntegration{Id: "GOOD"}
	mockFactory.EXPECT().CreateScanner(goodIntegration).Return(newFakeImageScanner(&fakeScanner{typ: "FAKE"}), nil).Times(2)

	badIntegration := &storage.ImageIntegration{Id: "BAD"}
	var nilFIS *fakeImageScanner
	mockFactory.EXPECT().CreateScanner(badIntegration).Return(nilFIS, errors.New(errText))

	err := s.UpdateImageIntegration(goodIntegration)
	require.Nil(t, err)
	assert.False(t, s.IsEmpty())

	err = s.UpdateImageIntegration(badIntegration)
	require.NotNil(t, err)
	assert.Equal(t, errText, err.Error())
	assert.False(t, s.IsEmpty())

	all := s.GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, "FAKE", all[0].GetScanner().Type())

	s.Clear()
	assert.True(t, s.IsEmpty())

	err = s.UpdateImageIntegration(goodIntegration)
	require.Nil(t, err)
	all = s.GetAll()
	assert.Len(t, all, 1)
	assert.Equal(t, "FAKE", all[0].GetScanner().Type())

	err = s.RemoveImageIntegration(goodIntegration.GetId())
	require.Nil(t, err)
	assert.True(t, s.IsEmpty())

}
