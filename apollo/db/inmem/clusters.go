package inmem

import (
	"fmt"
	"sort"
	"sync"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/proto"
)

type clusterStore struct {
	clusters map[string]*v1.Cluster
	lock     sync.Mutex

	persistent db.ClusterStorage
}

func newClusterStore(persistent db.ClusterStorage) *clusterStore {
	return &clusterStore{
		clusters:   make(map[string]*v1.Cluster),
		persistent: persistent,
	}
}

func (s *clusterStore) clone(cluster *v1.Cluster) *v1.Cluster {
	return proto.Clone(cluster).(*v1.Cluster)
}

func (s *clusterStore) loadFromPersistent() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	clusters, err := s.persistent.GetClusters()
	if err != nil {
		return err
	}
	for _, cluster := range clusters {
		s.clusters[cluster.Name] = cluster
	}
	return nil
}

// GetClusterResult retrieves a cluster by id
func (s *clusterStore) GetCluster(name string) (cluster *v1.Cluster, exists bool, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	cluster, exists = s.clusters[name]
	return s.clone(cluster), exists, nil
}

// GetClusterResults applies the filters from GetClusterResultsRequest and returns the Clusters
func (s *clusterStore) GetClusters() ([]*v1.Cluster, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	var clusters []*v1.Cluster
	for _, cluster := range s.clusters {
		clusters = append(clusters, s.clone(cluster))
	}
	sort.SliceStable(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})
	return clusters, nil
}

// AddCluster inserts a cluster into memory
func (s *clusterStore) AddCluster(cluster *v1.Cluster) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.clusters[cluster.Name]; ok {
		return fmt.Errorf("cluster %v already exists", cluster.Name)
	}
	if err := s.persistent.AddCluster(cluster); err != nil {
		return err
	}
	s.clusters[cluster.Name] = s.clone(cluster)
	return nil
}

func (s *clusterStore) UpdateCluster(cluster *v1.Cluster) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.persistent.UpdateCluster(cluster); err != nil {
		return err
	}
	s.clusters[cluster.Name] = s.clone(cluster)
	return nil
}

func (s *clusterStore) RemoveCluster(name string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if err := s.persistent.RemoveCluster(name); err != nil {
		return err
	}
	delete(s.clusters, name)
	return nil
}
