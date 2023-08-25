package declarativeconfig

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

type resourceWithTraitsMock struct {
	origin storage.Traits_Origin
}

func (m *resourceWithTraitsMock) GetTraits() *storage.Traits {
	return &storage.Traits{Origin: m.origin}
}

func TestContext(t *testing.T) {
	imperativeResource := &resourceWithTraitsMock{origin: storage.Traits_IMPERATIVE}
	declarativeResource := &resourceWithTraitsMock{origin: storage.Traits_DECLARATIVE}
	defaultResource := &resourceWithTraitsMock{origin: storage.Traits_DEFAULT}
	ctx := context.Background()
	declarativeCtx := WithModifyDeclarativeResource(ctx)
	// 1. empty context can modify imperative origin
	assert.True(t, CanModifyResource(ctx, imperativeResource))
	// 2. empty context can't modify declarative origin
	assert.False(t, CanModifyResource(ctx, declarativeResource))
	// 3. empty context can't modify default origin
	assert.False(t, CanModifyResource(ctx, defaultResource))
	// 4. context.WithModifyDeclarativeResource can modify declarative origin
	assert.True(t, CanModifyResource(declarativeCtx, declarativeResource))
	// 5. context.WithModifyDeclarativeResource can't modify imperative origin
	assert.False(t, CanModifyResource(declarativeCtx, imperativeResource))
	// 6. context.WithModifyDeclarativeResource can't modify default origin
	assert.False(t, CanModifyResource(declarativeCtx, defaultResource))
}
