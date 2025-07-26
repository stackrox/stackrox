package storagetov1

import (
	"testing"

	convertTestUtils "github.com/stackrox/rox/central/convert/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestTraits(t *testing.T) {
	traits := &storage.Traits{}
	require.NoError(t, testutils.FullInit(traits, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))

	v1Traits := Traits(traits)

	convertTestUtils.AssertProtoMessageEqual(t, traits, v1Traits)
}
