package ranking

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
)

// getID is a function that returns id of an object under given index.
type getID func(index int) string

// setPriority updates priority of object under given index.
type setPriority func(index int, priority int64)

// SetClustersPriorities obtain ranks for all elements and calculate and set their priorities.
func SetClustersPriorities(ranker *Ranker, elements []*storage.Cluster, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetImagesPriorities obtain ranks for all elements and calculate and set their priorities.
func SetImagesPriorities(ranker *Ranker, elements []*storage.Image, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetImageComponentsPriorities obtain ranks for all elements and calculate and set their priorities.
func SetImageComponentsPriorities(ranker *Ranker, elements []*storage.ImageComponent, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetNodesPriorities obtain ranks for all elements and calculate and set their priorities.
func SetNodesPriorities(ranker *Ranker, elements []*storage.Node, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetListImagesPriorities obtain ranks for all elements and calculate and set their priorities.
func SetListImagesPriorities(ranker *Ranker, elements []*storage.ListImage, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetNamespacesPriorities obtain ranks for all elements and calculate and set their priorities.
func SetNamespacesPriorities(ranker *Ranker, elements []*storage.NamespaceMetadata, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetDeploymentsPriorities obtain ranks for all elements and calculate and set their priorities.
func SetDeploymentsPriorities(ranker *Ranker, elements []*storage.Deployment, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// SetListDeploymentsPriorities obtain ranks for all elements and calculate and set their priorities.
func SetListDeploymentsPriorities(ranker *Ranker, elements []*storage.ListDeployment, offset int64) {
	setPriorities(ranker, len(elements), offset,
		func(i int) string { return elements[i].Id },
		func(i int, p int64) { elements[i].Priority = p })
}

// setPriorities sets priorities based on ranker response for Id under index.
// It iterates from 0 to length and uses setPriority to update priority under index.
// Priorities are strictly monotonic and fit in range [offset+1,offset+len(ids)].
// There is no guarantee for any particular order for elements with the same rank.
func setPriorities(ranker *Ranker, length int, offset int64, getID getID, setPriority setPriority) {
	type rankWithIndex struct {
		rank  int64
		index int
	}

	ranks := make([]rankWithIndex, length)
	for i := 0; i < length; i++ {
		ranks[i] = rankWithIndex{
			rank:  ranker.GetRankForID(getID(i)),
			index: i,
		}
	}

	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].rank < ranks[j].rank
	})

	for priority, rank := range ranks {
		setPriority(rank.index, int64(priority)+offset+1)
	}
}
