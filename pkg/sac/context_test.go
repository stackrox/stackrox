package sac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyAccessScopeCheckerCore(t *testing.T) {
	noCore := context.Background()
	assert.Equal(t, noCore, CopyAccessScopeCheckerCore(noCore, context.Background()))

	denyAll := CopyAccessScopeCheckerCore(context.Background(), WithNoAccess(context.Background()))
	assert.False(t, GlobalAccessScopeChecker(denyAll).IsAllowed(ClusterScopeKey("x")))

	allowAll := CopyAccessScopeCheckerCore(context.Background(), WithAllAccess(context.Background()))
	assert.True(t, GlobalAccessScopeChecker(allowAll).IsAllowed(ClusterScopeKey("x")))
}
