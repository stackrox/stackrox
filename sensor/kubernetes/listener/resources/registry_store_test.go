package resources

import (
	"testing"

	"github.com/stackrox/rox/pkg/docker/types"
	"github.com/stretchr/testify/assert"
)

func TestRegistryStore(t *testing.T) {
	rs := newRegistryStore()
	rs.addOrUpdateRegistry("a", "reg1", types.DockerConfigEntry{
		Username: "test1",
		Password: "test1pass",
		Email:    "test1@test.com",
	})
	rs.addOrUpdateRegistry("a", "reg2", types.DockerConfigEntry{
		Username: "test2",
		Password: "test2pass",
		Email:    "test2@test.com",
	})
	rs.addOrUpdateRegistry("b", "reg3", types.DockerConfigEntry{
		Username: "test3",
		Password: "test2pass",
		Email:    "test3@test.com",
	})

	regs := rs.getAllInNamespace("a")
	assert.Len(t, regs, 2)

	regs = rs.getAllInNamespace("b")
	assert.Len(t, regs, 1)

	regs = rs.getAllInNamespace("c")
	assert.Empty(t, regs)
}
