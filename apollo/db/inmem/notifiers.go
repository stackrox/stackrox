package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

type notifierStore struct {
	notifiers     map[string]*v1.Notifier
	notifierMutex sync.Mutex

	persistent db.NotifierStorage
}

func newNotifierStore(persistent db.NotifierStorage) *notifierStore {
	return &notifierStore{
		notifiers:  make(map[string]*v1.Notifier),
		persistent: persistent,
	}
}

func (s *notifierStore) loadFromPersistent() error {
	s.notifierMutex.Lock()
	defer s.notifierMutex.Unlock()
	notifiers, err := s.persistent.GetNotifiers(&v1.GetNotifiersRequest{})
	if err != nil {
		return err
	}
	for _, notifier := range notifiers {
		s.notifiers[notifier.Name] = notifier
	}
	return nil
}

// GetNotifierResult retrieves a notifier by id
func (s *notifierStore) GetNotifier(name string) (notifier *v1.Notifier, exists bool, err error) {
	s.notifierMutex.Lock()
	defer s.notifierMutex.Unlock()
	notifier, exists = s.notifiers[name]
	return
}

// GetNotifierResults applies the filters from GetNotifierResultsRequest and returns the Notifiers
func (s *notifierStore) GetNotifiers(request *v1.GetNotifiersRequest) ([]*v1.Notifier, error) {
	s.notifierMutex.Lock()
	defer s.notifierMutex.Unlock()
	var notifiers []*v1.Notifier
	for _, notifier := range s.notifiers {
		notifiers = append(notifiers, notifier)
	}
	sort.SliceStable(notifiers, func(i, j int) bool {
		return notifiers[i].Name < notifiers[j].Name
	})
	return notifiers, nil
}

// AddNotifier inserts a notifier into memory
func (s *notifierStore) AddNotifier(notifier *v1.Notifier) error {
	s.notifierMutex.Lock()
	defer s.notifierMutex.Unlock()
	if _, ok := s.notifiers[notifier.Name]; ok {
		return fmt.Errorf("notifier %v already exists", notifier.Name)
	}
	if err := s.persistent.AddNotifier(notifier); err != nil {
		return err
	}
	s.notifiers[notifier.Name] = notifier
	return nil
}

// UpdateNotifier updates a notifier
func (s *notifierStore) UpdateNotifier(notifier *v1.Notifier) error {
	s.notifierMutex.Lock()
	defer s.notifierMutex.Unlock()
	if err := s.persistent.UpdateNotifier(notifier); err != nil {
		return err
	}
	s.notifiers[notifier.Name] = notifier
	return nil
}

// RemoveNotifier deletes a notifier if it exists
func (s *notifierStore) RemoveNotifier(name string) error {
	s.notifierMutex.Lock()
	defer s.notifierMutex.Unlock()
	if err := s.persistent.RemoveNotifier(name); err != nil {
		return err
	}
	delete(s.notifiers, name)
	return nil
}
