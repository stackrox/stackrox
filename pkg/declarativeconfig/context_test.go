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
	dynamicResource := &resourceWithTraitsMock{origin: storage.Traits_DYNAMIC}
	declarativeResource := &resourceWithTraitsMock{origin: storage.Traits_DECLARATIVE}
	defaultResource := &resourceWithTraitsMock{origin: storage.Traits_DEFAULT}
	ctx := context.Background()
	declarativeCtx := WithModifyDeclarativeResource(ctx)
	declarativeOrImperativeCtx := WithModifyDeclarativeOrImperative(ctx)
	// 1.1. empty context can modify imperative origin
	assert.True(t, CanModifyResource(ctx, imperativeResource))
	// 1.2. empty context can modify dynamic origin
	assert.True(t, CanModifyResource(ctx, dynamicResource))
	// 2. empty context can't modify declarative origin
	assert.False(t, CanModifyResource(ctx, declarativeResource))
	// 3. empty context can't modify default origin
	assert.False(t, CanModifyResource(ctx, defaultResource))
	// 4. context.WithModifyDeclarativeResource can modify declarative origin
	assert.True(t, CanModifyResource(declarativeCtx, declarativeResource))
	// 5.1. context.WithModifyDeclarativeResource can't modify imperative origin
	assert.False(t, CanModifyResource(declarativeCtx, imperativeResource))
	// 5.2. context.WithModifyDeclarativeResource can't modify dynamic origin
	assert.False(t, CanModifyResource(declarativeCtx, dynamicResource))
	// 6. context.WithModifyDeclarativeResource can't modify default origin
	assert.False(t, CanModifyResource(declarativeCtx, defaultResource))
	// 7. context.WithModifyDeclarativeOrImperative can modify declarative origin
	assert.True(t, CanModifyResource(declarativeOrImperativeCtx, declarativeResource))
	// 8.1. context.WithModifyDeclarativeOrImperative can modify imperative origin
	assert.True(t, CanModifyResource(declarativeOrImperativeCtx, imperativeResource))
	// 8.2. context.WithModifyDeclarativeOrImperative can modify dynamic origin
	assert.True(t, CanModifyResource(declarativeOrImperativeCtx, dynamicResource))
	// 9. context.WithModifyDeclarativeOrImperative can't modify default origin
	assert.False(t, CanModifyResource(declarativeOrImperativeCtx, defaultResource))
}
