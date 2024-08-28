package concurrency

// WaitForAll waits until all the given waitables have been triggered at least once. It returns true if this has been
// observed, false if cancelCond is triggered beforehand.
func WaitForAll(cancelCond Waitable, waitables ...Waitable) bool {
	chans := make([]<-chan struct{}, len(waitables))
	for i, w := range waitables {
		chans[i] = w.Done()
	}

	for {
		if len(chans) == 0 {
			return true
		}

		select {
		case <-cancelCond.Done():
			return false
		case <-chans[0]:
		}
		chans = chans[1:]
	}
}
