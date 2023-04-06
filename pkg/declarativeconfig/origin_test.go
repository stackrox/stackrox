package declarativeconfig

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

type tstResourceWithTraits struct {
	traits *storage.Traits
}

func (r *tstResourceWithTraits) GetTraits() *storage.Traits {
	return r.traits
}

func TestVerifyReferencedResourceOrigin(t *testing.T) {
	declarativeTraits := &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	imperativeTraits := &storage.Traits{
		Origin: storage.Traits_IMPERATIVE,
	}
	orphanedTraits := &storage.Traits{
		Origin: storage.Traits_DECLARATIVE_ORPHANED,
	}
	// Let's not enforce rules for when default resource is referencing another resource.
	// There's no need for that since default resources are immutable and created on a system startup.
	defaultTraits := &storage.Traits{
		Origin: storage.Traits_DEFAULT,
	}
	testNoError(t, declarativeTraits, declarativeTraits)
	testNoError(t, declarativeTraits, orphanedTraits)
	testNoError(t, declarativeTraits, defaultTraits)
	testError(t, declarativeTraits, imperativeTraits)

	testNoError(t, orphanedTraits, declarativeTraits)
	testNoError(t, orphanedTraits, orphanedTraits)
	testNoError(t, orphanedTraits, defaultTraits)
	testError(t, orphanedTraits, imperativeTraits)

	testNoError(t, imperativeTraits, declarativeTraits)
	testNoError(t, imperativeTraits, orphanedTraits)
	testNoError(t, imperativeTraits, defaultTraits)
	testNoError(t, imperativeTraits, imperativeTraits)
}

func testError(t *testing.T, referencing *storage.Traits, referenced *storage.Traits) {
	referencingResource := &tstResourceWithTraits{
		traits: referencing,
	}
	referencedResource := &tstResourceWithTraits{
		traits: referenced,
	}
	err := VerifyReferencedResourceOrigin(referencedResource, referencingResource, "", "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, errox.InvalidArgs)
}

func testNoError(t *testing.T, referencing *storage.Traits, referenced *storage.Traits) {
	referencingResource := &tstResourceWithTraits{
		traits: referencing,
	}
	referencedResource := &tstResourceWithTraits{
		traits: referenced,
	}
	assert.NoError(t, VerifyReferencedResourceOrigin(referencedResource, referencingResource, "", ""))
}
