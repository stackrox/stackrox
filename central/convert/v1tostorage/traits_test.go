package v1tostorage

import (
	"testing"

	convertTestUtils "github.com/stackrox/rox/central/convert/testutils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestTraits(t *testing.T) {
	traits := &v1.Traits{}
	require.NoError(t, testutils.FullInit(traits, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	v1Traits := Traits(traits)

	convertTestUtils.AssertProtoMessageEqual(t, traits, v1Traits)
}
