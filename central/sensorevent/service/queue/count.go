package queue

// Count returns the number of items currently in the queue.
func (p *persistedEventQueue) Count() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// If there are no waiting events, simply return.
	return len(p.seqIDQueue)
}
