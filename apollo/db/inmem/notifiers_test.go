package inmem

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func testNotifiers(t *testing.T, insertStorage, retrievalStorage db.NotifierStorage) {
	notifiers := []*v1.Notifier{
		{
			Name:   "pagerduty1",
			Type:   "pagerduty",
			Config: map[string]string{"username": "srox"},
		},
		{
			Name:   "slack1",
			Type:   "slack",
			Config: map[string]string{"username": "srox"},
		},
	}

	// Test Add
	for _, b := range notifiers {
		assert.NoError(t, insertStorage.AddNotifier(b))
	}
	// Verify that adding a notifier that exists does not deadlock and fails
	for _, b := range notifiers {
		assert.Error(t, insertStorage.AddNotifier(b))
	}

	for _, b := range notifiers {
		got, exists, err := retrievalStorage.GetNotifier(b.Name)
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

	// Test Update
	for _, b := range notifiers {
		b.Config["param"] = "newparam"
	}

	for _, b := range notifiers {
		assert.NoError(t, insertStorage.UpdateNotifier(b))
	}

	for _, b := range notifiers {
		got, exists, err := retrievalStorage.GetNotifier(b.GetName())
		assert.NoError(t, err)
		assert.True(t, exists)
		assert.Equal(t, got, b)
	}

	// Test Remove
	for _, b := range notifiers {
		assert.NoError(t, insertStorage.RemoveNotifier(b.GetName()))
	}

	for _, b := range notifiers {
		_, exists, err := retrievalStorage.GetNotifier(b.GetName())
		assert.NoError(t, err)
		assert.False(t, exists)
	}

}

func TestNotifiersPersistence(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newNotifierStore(persistent)
	testNotifiers(t, storage, persistent)
}

func TestNotifiers(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newNotifierStore(persistent)
	testNotifiers(t, storage, storage)
}

func TestNotifiersFiltering(t *testing.T) {
	t.Parallel()
	persistent, err := createBoltDB()
	if err != nil {
		t.Fatal(err)
	}
	storage := newNotifierStore(persistent)

	notifiers := []*v1.Notifier{
		{
			Name:   "pagerduty1",
			Type:   "pagerduty",
			Config: map[string]string{"username": "srox"},
		},
		{
			Name:   "slack1",
			Type:   "slack",
			Config: map[string]string{"username": "srox"},
		},
	}

	// Test Add
	for _, r := range notifiers {
		assert.NoError(t, storage.AddNotifier(r))
	}

	actualNotifiers, err := storage.GetNotifiers(&v1.GetNotifiersRequest{})
	assert.NoError(t, err)
	assert.Equal(t, notifiers, actualNotifiers)
}
