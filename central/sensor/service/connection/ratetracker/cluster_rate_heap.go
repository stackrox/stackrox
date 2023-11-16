package ratetracker

// Keeps track of the leading consumer clusters, which serves as a reference
// for identifying potential candidates for rate limiting.
type clusterRatesHeap []*clusterRate

func (h *clusterRatesHeap) Len() int {
	return len(*h)
}

func (h *clusterRatesHeap) Less(i, j int) bool {
	return (*h)[i].ratePerSec < (*h)[j].ratePerSec
}

func (h *clusterRatesHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
	(*h)[i].index = i
	(*h)[j].index = j
}

func (h *clusterRatesHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*clusterRate)
	item.index = n
	*h = append(*h, item)
}

func (h *clusterRatesHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]

	return item
}
