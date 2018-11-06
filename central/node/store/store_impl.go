package store

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper/crud/proto"
)

type storeImpl struct {
	crud proto.MessageCrud
}

func (s *storeImpl) CountNodes() (int, error) {
	return s.crud.Count()
}

func (s *storeImpl) ListNodes() ([]*v1.Node, error) {
	entries, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	nodes := make([]*v1.Node, len(entries))
	for i, entry := range entries {
		nodes[i] = entry.(*v1.Node)
	}
	return nodes, nil
}

func (s *storeImpl) GetNode(id string) (*v1.Node, error) {
	value, err := s.crud.Read(id)
	if err != nil {
		return nil, err
	}
	return value.(*v1.Node), nil
}

func (s *storeImpl) UpsertNode(node *v1.Node) error {
	return s.crud.Upsert(node)
}

func (s *storeImpl) RemoveNode(id string) error {
	return s.crud.Delete(id)
}
