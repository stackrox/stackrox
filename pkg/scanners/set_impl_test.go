package scanners

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/generated/api/v1"
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
