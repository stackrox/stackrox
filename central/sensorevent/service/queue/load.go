package queue

// Load fills the queue from events currently stored in the provided storage instance.
func (p *persistedEventQueue) Load(clusterID string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var err error
	p.seqIDQueue, p.depIDToSeqID, err = p.eventStorage.GetSensorEventIds(clusterID)
	return err
}
