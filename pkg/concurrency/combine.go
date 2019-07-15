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

// All returns a waitable that is triggered once all the given waitables have been triggered at least once. If
// cancelCond is triggered before this happens, the returned waitable will never trigger.
func All(cancelCond Waitable, waitables ...Waitable) WaitableChan {
	if len(waitables) == 0 {
		return ClosedChannel()
	}

	ch := make(chan struct{})
	go func() {
		if WaitForAll(cancelCond, waitables...) {
			close(ch)
		}
	}()

	return ch
}

// Any returns a waitable that is triggered once any of the given waitables is triggered. If cancelCond is triggered
// before this happens, the returned waitable will never trigger.
func Any(cancelCond Waitable, waitables ...Waitable) WaitableChan {
	if len(waitables) == 0 {
		return Never()
	}

	sig := NewSignal()
	for _, w := range waitables {
		go sig.SignalWhen(w, cancelCond)
	}

	return sig.WaitC()
}

// WaitForAny waits until any of the given waitables is triggered. It returns true if this happens, false if cancelCond
// is triggered beforehand.
func WaitForAny(cancelCond Waitable, waitables ...Waitable) bool {
	return WaitInContext(Any(cancelCond, waitables...), cancelCond)
}
