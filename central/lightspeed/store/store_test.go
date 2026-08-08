package store

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	s := New()

	t.Run("SetHost and GetHost", func(t *testing.T) {
		s.SetHost("cluster1", "https://lightspeed.example.com")
		assert.Equal(t, "https://lightspeed.example.com", s.GetHost("cluster1"))
	})

	t.Run("GetHost for unknown cluster returns empty string", func(t *testing.T) {
		assert.Equal(t, "", s.GetHost("unknown"))
	})

	t.Run("UpdateInfo", func(t *testing.T) {
		info := &central.LightspeedInfo{
			Host:           "https://lightspeed.example.com",
			IsReady:        true,
			HasQueryAccess: true,
		}
		s.UpdateInfo("cluster1", info)

		host, gotInfo := s.Get("cluster1")
		assert.Equal(t, "https://lightspeed.example.com", host)
		assert.Equal(t, info, gotInfo)
	})

	t.Run("SetHost resets info to nil", func(t *testing.T) {
		info := &central.LightspeedInfo{
			Host:    "https://lightspeed.example.com",
			IsReady: true,
		}
		s.UpdateInfo("cluster2", info)

		_, gotInfo := s.Get("cluster2")
		assert.NotNil(t, gotInfo)

		s.SetHost("cluster2", "https://new-host.example.com")

		host, gotInfo := s.Get("cluster2")
		assert.Equal(t, "https://new-host.example.com", host)
		assert.Nil(t, gotInfo)
	})

	t.Run("UpdateInfo for cluster without host", func(t *testing.T) {
		info := &central.LightspeedInfo{
			Host:    "https://lightspeed.example.com",
			IsReady: true,
		}
		s.UpdateInfo("cluster3", info)

		host, gotInfo := s.Get("cluster3")
		assert.Equal(t, "", host)
		assert.Equal(t, info, gotInfo)
	})
}

func TestSingleton(t *testing.T) {
	s1 := Singleton()
	s2 := Singleton()

	assert.Same(t, s1, s2, "Singleton should return the same instance")
}
