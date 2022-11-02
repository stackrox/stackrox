// Package dag implements directed acyclic graphs (DAGs).
package dag

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
)

// IDInterface describes the interface a type must implement in order to
// explicitly specify vertex id.
//
// Objects of types not implementing this interface will receive automatically
// generated ids (as of adding them to the graph).
type IDInterface interface {
	ID() string
}

// DAG implements the data structure of the DAG.
type DAG struct {
	muDAG            sync.RWMutex
	vertices         map[interface{}]string
	vertexIds        map[string]interface{}
	inboundEdge      map[interface{}]map[interface{}]struct{}
	outboundEdge     map[interface{}]map[interface{}]struct{}
	muCache          sync.RWMutex
	verticesLocked   *dMutex
	ancestorsCache   map[interface{}]map[interface{}]struct{}
	descendantsCache map[interface{}]map[interface{}]struct{}
}

// NewDAG creates / initializes a new DAG.
func NewDAG() *DAG {
	return &DAG{
		vertices:         make(map[interface{}]string),
		vertexIds:        make(map[string]interface{}),
		inboundEdge:      make(map[interface{}]map[interface{}]struct{}),
		outboundEdge:     make(map[interface{}]map[interface{}]struct{}),
		verticesLocked:   newDMutex(),
		ancestorsCache:   make(map[interface{}]map[interface{}]struct{}),
		descendantsCache: make(map[interface{}]map[interface{}]struct{}),
	}
}

// AddVertex adds the vertex v to the DAG. AddVertex returns an error, if v is
// nil, v is already part of the graph, or the id of v is already part of the
// graph.
func (d *DAG) AddVertex(v interface{}) (string, error) {

	d.muDAG.Lock()
	defer d.muDAG.Unlock()

	return d.addVertex(v)
}

func (d *DAG) addVertex(v interface{}) (string, error) {

	var id string
	if i, ok := v.(IDInterface); ok {
		id = i.ID()
	} else {
		id = uuid.New().String()
	}

	err := d.addVertexByID(id, v)
	return id, err
}

// AddVertexByID adds the vertex v and the specified id to the DAG.
// AddVertexByID returns an error, if v is nil, v is already part of the graph,
// or the specified id is already part of the graph.
func (d *DAG) AddVertexByID(id string, v interface{}) error {

	d.muDAG.Lock()
	defer d.muDAG.Unlock()

	return d.addVertexByID(id, v)
}

func (d *DAG) addVertexByID(id string, v interface{}) error {

	// sanity checking
	if v == nil {
		return VertexNilError{}
	}
	if _, exists := d.vertices[v]; exists {
		return VertexDuplicateError{v}
	}

	if _, exists := d.vertexIds[id]; exists {
		return IDDuplicateError{id}
	}

	d.vertices[v] = id
	d.vertexIds[id] = v

	return nil
}

// GetVertex returns a vertex by its id. GetVertex returns an error, if id is
// the empty string or unknown.
func (d *DAG) GetVertex(id string) (interface{}, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	if id == "" {
		return nil, IDEmptyError{}
	}

	v, exists := d.vertexIds[id]
	if !exists {
		return nil, IDUnknownError{id}
	}
	return v, nil
}

// DeleteVertex deletes the vertex with the given id. DeleteVertex also
// deletes all attached edges (inbound and outbound). DeleteVertex returns
// an error, if id is empty or unknown.
func (d *DAG) DeleteVertex(id string) error {

	d.muDAG.Lock()
	defer d.muDAG.Unlock()

	if err := d.saneID(id); err != nil {
		return err
	}

	v := d.vertexIds[id]

	// get descendents and ancestors as they are now
	descendants := copyMap(d.getDescendants(v))
	ancestors := copyMap(d.getAncestors(v))

	// delete v in outbound edges of parents
	if _, exists := d.inboundEdge[v]; exists {
		for parent := range d.inboundEdge[v] {
			delete(d.outboundEdge[parent], v)
		}
	}

	// delete v in inbound edges of children
	if _, exists := d.outboundEdge[v]; exists {
		for child := range d.outboundEdge[v] {
			delete(d.inboundEdge[child], v)
		}
	}

	// delete in- and outbound of v itself
	delete(d.inboundEdge, v)
	delete(d.outboundEdge, v)

	// for v and all its descendants delete cached ancestors
	for descendant := range descendants {
		delete(d.ancestorsCache, descendant)
	}
	delete(d.ancestorsCache, v)

	// for v and all its ancestors delete cached descendants
	for ancestor := range ancestors {
		delete(d.descendantsCache, ancestor)
	}
	delete(d.descendantsCache, v)

	// delete v itself
	delete(d.vertices, v)
	delete(d.vertexIds, id)

	return nil
}

// AddEdge adds an edge between srcID and dstID. AddEdge returns an
// error, if srcID or dstID are empty strings or unknown, if the edge
// already exists, or if the new edge would create a loop.
func (d *DAG) AddEdge(srcID, dstID string) error {

	d.muDAG.Lock()
	defer d.muDAG.Unlock()

	if err := d.saneID(srcID); err != nil {
		return err
	}

	if err := d.saneID(dstID); err != nil {
		return err
	}

	if srcID == dstID {
		return SrcDstEqualError{srcID, dstID}
	}

	src := d.vertexIds[srcID]
	dst := d.vertexIds[dstID]

	// if the edge is already known, there is nothing else to do
	if d.isEdge(src, dst) {
		return EdgeDuplicateError{srcID, dstID}
	}

	// get descendents and ancestors as they are now
	descendants := copyMap(d.getDescendants(dst))
	ancestors := copyMap(d.getAncestors(src))

	if _, exists := descendants[src]; exists {
		return EdgeLoopError{srcID, dstID}
	}

	// prepare d.outbound[src], iff needed
	if _, exists := d.outboundEdge[src]; !exists {
		d.outboundEdge[src] = make(map[interface{}]struct{})
	}

	// dst is a child of src
	d.outboundEdge[src][dst] = struct{}{}

	// prepare d.inboundEdge[dst], iff needed
	if _, exists := d.inboundEdge[dst]; !exists {
		d.inboundEdge[dst] = make(map[interface{}]struct{})
	}

	// src is a parent of dst
	d.inboundEdge[dst][src] = struct{}{}

	// for dst and all its descendants delete cached ancestors
	for descendant := range descendants {
		delete(d.ancestorsCache, descendant)
	}
	delete(d.ancestorsCache, dst)

	// for src and all its ancestors delete cached descendants
	for ancestor := range ancestors {
		delete(d.descendantsCache, ancestor)
	}
	delete(d.descendantsCache, src)

	return nil
}

// IsEdge returns true, if there exists an edge between srcID and dstID.
// IsEdge returns false, if there is no such edge. IsEdge returns an error,
// if srcID or dstID are empty, unknown, or the same.
func (d *DAG) IsEdge(srcID, dstID string) (bool, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	if err := d.saneID(srcID); err != nil {
		return false, err
	}
	if err := d.saneID(dstID); err != nil {
		return false, err
	}
	if srcID == dstID {
		return false, SrcDstEqualError{srcID, dstID}
	}

	return d.isEdge(d.vertexIds[srcID], d.vertexIds[dstID]), nil
}

func (d *DAG) isEdge(src, dst interface{}) bool {

	if _, exists := d.outboundEdge[src]; !exists {
		return false
	}
	if _, exists := d.outboundEdge[src][dst]; !exists {
		return false
	}
	if _, exists := d.inboundEdge[dst]; !exists {
		return false
	}
	if _, exists := d.inboundEdge[dst][src]; !exists {
		return false
	}
	return true
}

// DeleteEdge deletes the edge between srcID and dstID. DeleteEdge
// returns an error, if srcID or dstID are empty or unknown, or if,
// there is no edge between srcID and dstID.
func (d *DAG) DeleteEdge(srcID, dstID string) error {

	d.muDAG.Lock()
	defer d.muDAG.Unlock()

	if err := d.saneID(srcID); err != nil {
		return err
	}
	if err := d.saneID(dstID); err != nil {
		return err
	}
	if srcID == dstID {
		return SrcDstEqualError{srcID, dstID}
	}

	src := d.vertexIds[srcID]
	dst := d.vertexIds[dstID]

	if !d.isEdge(src, dst) {
		return EdgeUnknownError{srcID, dstID}
	}

	// get descendents and ancestors as they are now
	descendants := copyMap(d.getDescendants(src))
	ancestors := copyMap(d.getAncestors(dst))

	// delete outbound and inbound
	delete(d.outboundEdge[src], dst)
	delete(d.inboundEdge[dst], src)

	// for src and all its descendants delete cached ancestors
	for descendant := range descendants {
		delete(d.ancestorsCache, descendant)
	}
	delete(d.ancestorsCache, src)

	// for dst and all its ancestors delete cached descendants
	for ancestor := range ancestors {
		delete(d.descendantsCache, ancestor)
	}
	delete(d.descendantsCache, dst)

	return nil
}

// GetOrder returns the number of vertices in the graph.
func (d *DAG) GetOrder() int {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	return d.getOrder()
}

func (d *DAG) getOrder() int {
	return len(d.vertices)
}

// GetSize returns the number of edges in the graph.
func (d *DAG) GetSize() int {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	return d.getSize()
}

func (d *DAG) getSize() int {
	count := 0
	for _, value := range d.outboundEdge {
		count += len(value)
	}
	return count
}

// GetLeaves returns all vertices without children.
func (d *DAG) GetLeaves() map[string]interface{} {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	return d.getLeaves()
}

func (d *DAG) getLeaves() map[string]interface{} {
	leaves := make(map[string]interface{})
	for v := range d.vertices {
		dstIDs, ok := d.outboundEdge[v]
		if !ok || len(dstIDs) == 0 {
			id := d.vertices[v]
			leaves[id] = v
		}
	}
	return leaves
}

// IsLeaf returns true, if the vertex with the given id has no children. IsLeaf
// returns an error, if id is empty or unknown.
func (d *DAG) IsLeaf(id string) (bool, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	if err := d.saneID(id); err != nil {
		return false, err
	}
	return d.isLeaf(id), nil
}

func (d *DAG) isLeaf(id string) bool {
	v := d.vertexIds[id]
	dstIDs, ok := d.outboundEdge[v]
	if !ok || len(dstIDs) == 0 {
		return true
	}
	return false
}

// GetRoots returns all vertices without parents.
func (d *DAG) GetRoots() map[string]interface{} {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	return d.getRoots()
}

func (d *DAG) getRoots() map[string]interface{} {
	roots := make(map[string]interface{})
	for v := range d.vertices {
		srcIDs, ok := d.inboundEdge[v]
		if !ok || len(srcIDs) == 0 {
			id := d.vertices[v]
			roots[id] = v
		}
	}
	return roots
}

// IsRoot returns true, if the vertex with the given id has no parents. IsRoot
// returns an error, if id is empty or unknown.
func (d *DAG) IsRoot(id string) (bool, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	if err := d.saneID(id); err != nil {
		return false, err
	}
	return d.isRoot(id), nil
}

func (d *DAG) isRoot(id string) bool {
	v := d.vertexIds[id]
	srcIDs, ok := d.inboundEdge[v]
	if !ok || len(srcIDs) == 0 {
		return true
	}
	return false
}

// GetVertices returns all vertices.
func (d *DAG) GetVertices() map[string]interface{} {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	out := make(map[string]interface{})
	for id, value := range d.vertexIds {
		out[id] = value
	}
	return out
}

// GetParents returns the all parents of the vertex with the id
// id. GetParents returns an error, if id is empty or unknown.
func (d *DAG) GetParents(id string) (map[string]interface{}, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	if err := d.saneID(id); err != nil {
		return nil, err
	}
	v := d.vertexIds[id]
	parents := make(map[string]interface{})
	for pv := range d.inboundEdge[v] {
		pid := d.vertices[pv]
		parents[pid] = pv
	}
	return parents, nil
}

// GetChildren returns all children of the vertex with the id
// id. GetChildren returns an error, if id is empty or unknown.
func (d *DAG) GetChildren(id string) (map[string]interface{}, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	return d.getChildren(id)
}

func (d *DAG) getChildren(id string) (map[string]interface{}, error) {
	if err := d.saneID(id); err != nil {
		return nil, err
	}
	v := d.vertexIds[id]
	children := make(map[string]interface{})
	for cv := range d.outboundEdge[v] {
		cid := d.vertices[cv]
		children[cid] = cv
	}
	return children, nil
}

// GetAncestors return all ancestors of the vertex with the id id. GetAncestors
// returns an error, if id is empty or unknown.
//
// Note, in order to get the ancestors, GetAncestors populates the ancestor-
// cache as needed. Depending on order and size of the sub-graph of the vertex
// with id id this may take a long time and consume a lot of memory.
func (d *DAG) GetAncestors(id string) (map[string]interface{}, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	if err := d.saneID(id); err != nil {
		return nil, err
	}
	v := d.vertexIds[id]
	ancestors := make(map[string]interface{})
	for av := range d.getAncestors(v) {
		aid := d.vertices[av]
		ancestors[aid] = av
	}
	return ancestors, nil
}

func (d *DAG) getAncestors(v interface{}) map[interface{}]struct{} {

	// in the best case we have already a populated cache
	d.muCache.RLock()
	cache, exists := d.ancestorsCache[v]
	d.muCache.RUnlock()
	if exists {
		return cache
	}

	// lock this vertex to work on it exclusively
	d.verticesLocked.lock(v)
	defer d.verticesLocked.unlock(v)

	// now as we have locked this vertex, check (again) that no one has
	// meanwhile populated the cache
	d.muCache.RLock()
	cache, exists = d.ancestorsCache[v]
	d.muCache.RUnlock()
	if exists {
		return cache
	}

	// as there is no cache, we start from scratch and collect all ancestors locally
	cache = make(map[interface{}]struct{})
	var mu sync.Mutex
	if parents, ok := d.inboundEdge[v]; ok {

		// for each parent collect its ancestors
		for parent := range parents {
			parentAncestors := d.getAncestors(parent)
			mu.Lock()
			for ancestor := range parentAncestors {
				cache[ancestor] = struct{}{}
			}
			cache[parent] = struct{}{}
			mu.Unlock()
		}
	}

	// remember the collected descendents
	d.muCache.Lock()
	d.ancestorsCache[v] = cache
	d.muCache.Unlock()
	return cache
}

// GetOrderedAncestors returns all ancestors of the vertex with id id
// in a breath-first order. Only the first occurrence of each vertex is
// returned. GetOrderedAncestors returns an error, if id is empty or
// unknown.
//
// Note, there is no order between sibling vertices. Two consecutive runs of
// GetOrderedAncestors may return different results.
func (d *DAG) GetOrderedAncestors(id string) ([]string, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	ids, _, err := d.AncestorsWalker(id)
	if err != nil {
		return nil, err
	}
	var ancestors []string
	for aid := range ids {
		ancestors = append(ancestors, aid)
	}
	return ancestors, nil
}

// AncestorsWalker returns a channel and subsequently returns / walks all
// ancestors of the vertex with id id in a breath first order. The second
// channel returned may be used to stop further walking. AncestorsWalker
// returns an error, if id is empty or unknown.
//
// Note, there is no order between sibling vertices. Two consecutive runs of
// AncestorsWalker may return different results.
func (d *DAG) AncestorsWalker(id string) (chan string, chan bool, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	if err := d.saneID(id); err != nil {
		return nil, nil, err
	}
	ids := make(chan string)
	signal := make(chan bool, 1)
	go func() {
		d.muDAG.RLock()
		v := d.vertexIds[id]
		d.walkAncestors(v, ids, signal)
		d.muDAG.RUnlock()
		close(ids)
		close(signal)
	}()
	return ids, signal, nil
}

func (d *DAG) walkAncestors(v interface{}, ids chan string, signal chan bool) {

	var fifo []interface{}
	visited := make(map[interface{}]struct{})
	for parent := range d.inboundEdge[v] {
		visited[parent] = struct{}{}
		fifo = append(fifo, parent)
	}
	for {
		if len(fifo) == 0 {
			return
		}
		top := fifo[0]
		fifo = fifo[1:]
		for parent := range d.inboundEdge[top] {
			if _, exists := visited[parent]; !exists {
				visited[parent] = struct{}{}
				fifo = append(fifo, parent)
			}
		}
		select {
		case <-signal:
			return
		default:
			ids <- d.vertices[top]
		}
	}
}

// GetDescendants return all descendants of the vertex with id id.
// GetDescendants returns an error, if id is empty or unknown.
//
// Note, in order to get the descendants, GetDescendants populates the
// descendants-cache as needed. Depending on order and size of the sub-graph
// of the vertex with id id this may take a long time and consume a lot
// of memory.
func (d *DAG) GetDescendants(id string) (map[string]interface{}, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	if err := d.saneID(id); err != nil {
		return nil, err
	}
	v := d.vertexIds[id]
	//return copyMap(d.getAncestors(v)), nil

	descendants := make(map[string]interface{})
	for dv := range d.getDescendants(v) {
		did := d.vertices[dv]
		descendants[did] = dv
	}
	return descendants, nil
}

func (d *DAG) getDescendants(v interface{}) map[interface{}]struct{} {

	// in the best case we have already a populated cache
	d.muCache.RLock()
	cache, exists := d.descendantsCache[v]
	d.muCache.RUnlock()
	if exists {
		return cache
	}

	// lock this vertex to work on it exclusively
	d.verticesLocked.lock(v)
	defer d.verticesLocked.unlock(v)

	// now as we have locked this vertex, check (again) that no one has
	// meanwhile populated the cache
	d.muCache.RLock()
	cache, exists = d.descendantsCache[v]
	d.muCache.RUnlock()
	if exists {
		return cache
	}

	// as there is no cache, we start from scratch and collect all descendants
	// locally
	cache = make(map[interface{}]struct{})
	var mu sync.Mutex
	if children, ok := d.outboundEdge[v]; ok {

		// for each child use a goroutine to collect its descendants
		//var waitGroup sync.WaitGroup
		//waitGroup.Add(len(children))
		for child := range children {
			//go func(child interface{}, mu *sync.Mutex, cache map[interface{}]bool) {
			childDescendants := d.getDescendants(child)
			mu.Lock()
			for descendant := range childDescendants {
				cache[descendant] = struct{}{}
			}
			cache[child] = struct{}{}
			mu.Unlock()
			//waitGroup.Done()
			//}(child, &mu, cache)
		}
		//waitGroup.Wait()
	}

	// remember the collected descendents
	d.muCache.Lock()
	d.descendantsCache[v] = cache
	d.muCache.Unlock()
	return cache
}

// GetOrderedDescendants returns all descendants of the vertex with id id
// in a breath-first order. Only the first occurrence of each vertex is
// returned. GetOrderedDescendants returns an error, if id is empty or
// unknown.
//
// Note, there is no order between sibling vertices. Two consecutive runs of
// GetOrderedDescendants may return different results.
func (d *DAG) GetOrderedDescendants(id string) ([]string, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	ids, _, err := d.DescendantsWalker(id)
	if err != nil {
		return nil, err
	}
	var descendants []string
	for did := range ids {
		descendants = append(descendants, did)
	}
	return descendants, nil
}

// GetDescendantsGraph returns a new DAG consisting of the vertex with id id and
// all its descendants (i.e. the subgraph). GetDescendantsGraph also returns the
// id of the (copy of the) given vertex within the new graph (i.e. the id of the
// single root of the new graph). GetDescendantsGraph returns an error, if id is
// empty or unknown.
//
// Note, the new graph is a copy of the relevant part of the original graph.
func (d *DAG) GetDescendantsGraph(id string) (*DAG, string, error) {

	// recursively add the current vertex and all its descendants
	return d.getRelativesGraph(id, false)
}

// GetAncestorsGraph returns a new DAG consisting of the vertex with id id and
// all its ancestors (i.e. the subgraph). GetAncestorsGraph also returns the id
// of the (copy of the) given vertex within the new graph (i.e. the id of the
// single leaf of the new graph). GetAncestorsGraph returns an error, if id is
// empty or unknown.
//
// Note, the new graph is a copy of the relevant part of the original graph.
func (d *DAG) GetAncestorsGraph(id string) (*DAG, string, error) {

	// recursively add the current vertex and all its ancestors
	return d.getRelativesGraph(id, true)
}

func (d *DAG) getRelativesGraph(id string, asc bool) (*DAG, string, error) {
	// sanity checking
	if id == "" {
		return nil, "", IDEmptyError{}
	}
	v, exists := d.vertexIds[id]
	if !exists {
		return nil, "", IDUnknownError{id}
	}

	// create a new dag
	newDAG := NewDAG()

	// protect the graph from modification
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	// recursively add the current vertex and all its relatives
	newId, err := d.getRelativesGraphRec(v, newDAG, make(map[interface{}]string), asc)
	return newDAG, newId, err
}

func (d *DAG) getRelativesGraphRec(v interface{}, newDAG *DAG, visited map[interface{}]string, asc bool) (newId string, err error) {

	// copy this vertex to the new graph
	if newId, err = newDAG.AddVertex(v); err != nil {
		return
	}

	// mark this vertex as visited
	visited[v] = newId

	// get the direct relatives (depending on the direction either parents or children)
	var relatives map[interface{}]struct{}
	var ok bool
	if asc {
		relatives, ok = d.inboundEdge[v]
	} else {
		relatives, ok = d.outboundEdge[v]
	}

	// for all direct relatives in the original graph
	if ok {
		for relative := range relatives {

			// if we haven't seen this relative
			relativeId, exists := visited[relative]
			if !exists {

				// recursively add this relative
				if relativeId, err = d.getRelativesGraphRec(relative, newDAG, visited, asc); err != nil {
					return
				}
			}

			// add edge to this relative (depending on the direction)
			var srcID, dstID string
			if asc {
				srcID, dstID = relativeId, newId

			} else {
				srcID, dstID = newId, relativeId
			}
			if err = newDAG.AddEdge(srcID, dstID); err != nil {
				return
			}
		}
	}
	return
}

// DescendantsWalker returns a channel and subsequently returns / walks all
// descendants of the vertex with id id in a breath first order. The second
// channel returned may be used to stop further walking. DescendantsWalker
// returns an error, if id is empty or unknown.
//
// Note, there is no order between sibling vertices. Two consecutive runs of
// DescendantsWalker may return different results.
func (d *DAG) DescendantsWalker(id string) (chan string, chan bool, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()
	if err := d.saneID(id); err != nil {
		return nil, nil, err
	}
	ids := make(chan string)
	signal := make(chan bool, 1)
	go func() {
		d.muDAG.RLock()
		v := d.vertexIds[id]
		d.walkDescendants(v, ids, signal)
		d.muDAG.RUnlock()
		close(ids)
		close(signal)
	}()
	return ids, signal, nil
}

func (d *DAG) walkDescendants(v interface{}, ids chan string, signal chan bool) {
	var fifo []interface{}
	visited := make(map[interface{}]struct{})
	for child := range d.outboundEdge[v] {
		visited[child] = struct{}{}
		fifo = append(fifo, child)
	}
	for {
		if len(fifo) == 0 {
			return
		}
		top := fifo[0]
		fifo = fifo[1:]
		for child := range d.outboundEdge[top] {
			if _, exists := visited[child]; !exists {
				visited[child] = struct{}{}
				fifo = append(fifo, child)
			}
		}
		select {
		case <-signal:
			return
		default:
			ids <- d.vertices[top]
		}
	}
}

// FlowResult describes the data to be passed between vertices in a DescendantsFlow.
type FlowResult struct {

	// The id of the vertex that produced this result.
	ID string

	// The actual result.
	Result interface{}

	// Any error. Note, DescendantsFlow does not care about this error. It is up to
	// the FlowCallback of downstream vertices to handle the error as needed - if
	// needed.
	Error error
}

// FlowCallback is the signature of the (callback-) function to call for each
// vertex within a DescendantsFlow, after all its parents have finished their
// work. The parameters of the function are the (complete) DAG, the current
// vertex ID, and the results of all its parents. An instance of FlowCallback
// should return a result or an error.
type FlowCallback func(d *DAG, id string, parentResults []FlowResult) (interface{}, error)

// DescendantsFlow traverses descendants of the vertex with the ID startID. For
// the vertex itself and each of its descendant it executes the given (callback-)
// function providing it the results of its respective parents. The (callback-)
// function is only executed after all parents have finished their work.
func (d *DAG) DescendantsFlow(startID string, inputs []FlowResult, callback FlowCallback) ([]FlowResult, error) {
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	// Get IDs of all descendant vertices.
	flowIDs, errDes := d.GetDescendants(startID)
	if errDes != nil {
		return []FlowResult{}, errDes
	}

	// inputChannels provides for input channels for each of the descendant vertices (+ the start-vertex).
	inputChannels := make(map[string]chan FlowResult, len(flowIDs)+1)

	// Iterate vertex IDs and create an input channel for each of them and a single
	// output channel for leaves. Note, this "pre-flight" is needed to ensure we
	// really have an input channel regardless of how we traverse the tree and spawn
	// workers.
	leafCount := 0
	for id := range flowIDs {

		// Get all parents of this vertex.
		parents, errPar := d.GetParents(id)
		if errPar != nil {
			return []FlowResult{}, errPar
		}

		// Create a buffered input channel that has capacity for all parent results.
		inputChannels[id] = make(chan FlowResult, len(parents))

		if d.isLeaf(id) {
			leafCount += 1
		}
	}

	// outputChannel caries the results of leaf vertices.
	outputChannel := make(chan FlowResult, leafCount)

	// To also process the start vertex and to have its results being passed to its
	// children, add it to the vertex IDs. Also add an input channel for the start
	// vertex and feed the inputs to this channel.
	flowIDs[startID] = struct{}{}
	inputChannels[startID] = make(chan FlowResult, len(inputs))
	for _, i := range inputs {
		inputChannels[startID] <- i
	}

	wg := sync.WaitGroup{}

	// Iterate all vertex IDs (now incl. start vertex) and handle each worker (incl.
	// inputs and outputs) in a separate goroutine.
	for id := range flowIDs {

		// Get all children of this vertex that later need to be notified. Note, we
		// collect all children before the goroutine to be able to release the read
		// lock as early as possible.
		children, errChildren := d.GetChildren(id)
		if errChildren != nil {
			return []FlowResult{}, errChildren
		}

		// Remember to wait for this goroutine.
		wg.Add(1)

		go func(id string) {

			// Get this vertex's input channel.
			// Note, only concurrent read here, which is fine.
			c := inputChannels[id]

			// Await all parent inputs and stuff them into a slice.
			parentCount := cap(c)
			parentResults := make([]FlowResult, parentCount)
			for i := 0; i < parentCount; i++ {
				parentResults[i] = <-c
			}

			// Execute the worker.
			result, errWorker := callback(d, id, parentResults)

			// Wrap the worker's result into a FlowResult.
			flowResult := FlowResult{
				ID:     id,
				Result: result,
				Error:  errWorker,
			}

			// Send this worker's FlowResult onto all children's input channels or, if it is
			// a leaf (i.e. no children), send the result onto the output channel.
			if len(children) > 0 {
				for child := range children {
					inputChannels[child] <- flowResult
				}
			} else {
				outputChannel <- flowResult
			}

			// "Sign off".
			wg.Done()

		}(id)
	}

	// Wait for all go routines to finish.
	wg.Wait()

	// Await all leaf vertex results and stuff them into a slice.
	resultCount := cap(outputChannel)
	results := make([]FlowResult, resultCount)
	for i := 0; i < resultCount; i++ {
		results[i] = <-outputChannel
	}

	return results, nil
}

// ReduceTransitively transitively reduce the graph.
//
// Note, in order to do the reduction the descendant-cache of all vertices is
// populated (i.e. the transitive closure). Depending on order and size of DAG
// this may take a long time and consume a lot of memory.
func (d *DAG) ReduceTransitively() {

	d.muDAG.Lock()
	defer d.muDAG.Unlock()

	graphChanged := false

	// populate the descendents cache for all roots (i.e. the whole graph)
	for _, root := range d.getRoots() {
		_ = d.getDescendants(root)
	}

	// for each vertex
	for v := range d.vertices {

		// map of descendants of the children of v
		descendentsOfChildrenOfV := make(map[interface{}]struct{})

		// for each child of v
		for childOfV := range d.outboundEdge[v] {

			// collect child descendants
			for descendent := range d.descendantsCache[childOfV] {
				descendentsOfChildrenOfV[descendent] = struct{}{}
			}
		}

		// for each child of v
		for childOfV := range d.outboundEdge[v] {

			// remove the edge between v and child, iff child is a
			// descendant of any of the children of v
			if _, exists := descendentsOfChildrenOfV[childOfV]; exists {
				delete(d.outboundEdge[v], childOfV)
				delete(d.inboundEdge[childOfV], v)
				graphChanged = true
			}
		}
	}

	// flush the descendants- and ancestor cache if the graph has changed
	if graphChanged {
		d.flushCaches()
	}
}

// FlushCaches completely flushes the descendants- and ancestor cache.
//
// Note, the only reason to call this method is to free up memory.
// Normally the caches are automatically maintained.
func (d *DAG) FlushCaches() {
	d.muDAG.Lock()
	defer d.muDAG.Unlock()
	d.flushCaches()
}

func (d *DAG) flushCaches() {
	d.ancestorsCache = make(map[interface{}]map[interface{}]struct{})
	d.descendantsCache = make(map[interface{}]map[interface{}]struct{})
}

// Copy returns a copy of the DAG.
func (d *DAG) Copy() (newDAG *DAG, err error) {

	// create a new dag
	newDAG = NewDAG()

	// create a map of visited vertices
	visited := make(map[interface{}]string)

	// protect the graph from modification
	d.muDAG.RLock()
	defer d.muDAG.RUnlock()

	// add all roots and their descendants to the new DAG
	for _, root := range d.GetRoots() {
		if _, err = d.getRelativesGraphRec(root, newDAG, visited, false); err != nil {
			return
		}
	}
	return
}

// String returns a textual representation of the graph.
func (d *DAG) String() string {
	result := fmt.Sprintf("DAG Vertices: %d - Edges: %d\n", d.GetOrder(), d.GetSize())
	result += "Vertices:\n"
	d.muDAG.RLock()
	for k := range d.vertices {
		result += fmt.Sprintf("  %v\n", k)
	}
	result += "Edges:\n"
	for v, children := range d.outboundEdge {
		for child := range children {
			result += fmt.Sprintf("  %v -> %v\n", v, child)
		}
	}
	d.muDAG.RUnlock()
	return result
}

func (d *DAG) saneID(id string) error {
	// sanity checking
	if id == "" {
		return IDEmptyError{}
	}
	_, exists := d.vertexIds[id]
	if !exists {
		return IDUnknownError{id}
	}
	return nil
}

func copyMap(in map[interface{}]struct{}) map[interface{}]struct{} {
	out := make(map[interface{}]struct{})
	for id, value := range in {
		out[id] = value
	}
	return out
}

/***************************
********** Errors **********
****************************/

// VertexNilError is the error type to describe the situation, that a nil is
// given instead of a vertex.
type VertexNilError struct{}

// Implements the error interface.
func (e VertexNilError) Error() string {
	return "don't know what to do with 'nil'"
}

// VertexDuplicateError is the error type to describe the situation, that a
// given vertex already exists in the graph.
type VertexDuplicateError struct {
	v interface{}
}

// Implements the error interface.
func (e VertexDuplicateError) Error() string {
	return fmt.Sprintf("'%v' is already known", e.v)
}

// IDDuplicateError is the error type to describe the situation, that a given
// vertex id already exists in the graph.
type IDDuplicateError struct {
	id string
}

// Implements the error interface.
func (e IDDuplicateError) Error() string {
	return fmt.Sprintf("the id '%s' is already known", e.id)
}

// IDEmptyError is the error type to describe the situation, that an empty
// string is given instead of a valid id.
type IDEmptyError struct{}

// Implements the error interface.
func (e IDEmptyError) Error() string {
	return "don't know what to do with \"\""
}

// IDUnknownError is the error type to describe the situation, that a given
// vertex does not exit in the graph.
type IDUnknownError struct {
	id string
}

// Implements the error interface.
func (e IDUnknownError) Error() string {
	return fmt.Sprintf("'%s' is unknown", e.id)
}

// EdgeDuplicateError is the error type to describe the situation, that an edge
// already exists in the graph.
type EdgeDuplicateError struct {
	src string
	dst string
}

// Implements the error interface.
func (e EdgeDuplicateError) Error() string {
	return fmt.Sprintf("edge between '%s' and '%s' is already known", e.src, e.dst)
}

// EdgeUnknownError is the error type to describe the situation, that a given
// edge does not exit in the graph.
type EdgeUnknownError struct {
	src string
	dst string
}

// Implements the error interface.
func (e EdgeUnknownError) Error() string {
	return fmt.Sprintf("edge between '%s' and '%s' is unknown", e.src, e.dst)
}

// EdgeLoopError is the error type to describe loop errors (i.e. errors that
// where raised to prevent establishing loops in the graph).
type EdgeLoopError struct {
	src string
	dst string
}

// Implements the error interface.
func (e EdgeLoopError) Error() string {
	return fmt.Sprintf("edge between '%s' and '%s' would create a loop", e.src, e.dst)
}

// SrcDstEqualError is the error type to describe the situation, that src and
// dst are equal.
type SrcDstEqualError struct {
	src string
	dst string
}

// Implements the error interface.
func (e SrcDstEqualError) Error() string {
	return fmt.Sprintf("src ('%s') and dst ('%s') equal", e.src, e.dst)
}

/***************************
********** dMutex **********
****************************/

type cMutex struct {
	mutex sync.Mutex
	count int
}

// Structure for dynamic mutexes.
type dMutex struct {
	mutexes     map[interface{}]*cMutex
	globalMutex sync.Mutex
}

// Initialize a new dynamic mutex structure.
func newDMutex() *dMutex {
	return &dMutex{
		mutexes: make(map[interface{}]*cMutex),
	}
}

// Get a lock for instance i
func (d *dMutex) lock(i interface{}) {

	// acquire global lock
	d.globalMutex.Lock()

	// if there is no cMutex for i, create it
	if _, ok := d.mutexes[i]; !ok {
		d.mutexes[i] = new(cMutex)
	}

	// increase the count in order to show, that we are interested in this
	// instance mutex (thus now one deletes it)
	d.mutexes[i].count++

	// remember the mutex for later
	mutex := &d.mutexes[i].mutex

	// as the cMutex is there, we have increased the count, and we know the
	// instance mutex, we can release the global lock
	d.globalMutex.Unlock()

	// and wait on the instance mutex
	(*mutex).Lock()
}

// Release the lock for instance i.
func (d *dMutex) unlock(i interface{}) {

	// acquire global lock
	d.globalMutex.Lock()

	// unlock instance mutex
	d.mutexes[i].mutex.Unlock()

	// decrease the count, as we are no longer interested in this instance
	// mutex
	d.mutexes[i].count--

	// if we are the last one interested in this instance mutex delete the
	// cMutex
	if d.mutexes[i].count == 0 {
		delete(d.mutexes, i)
	}

	// release the global lock
	d.globalMutex.Unlock()
}
