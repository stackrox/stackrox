package crud

// DeleterOption represents an option on a created Deleter.
type DeleterOption func(*deleterImpl)

// GCAllChildren makes the deleter remove all children referenced from the deleted key.
func GCAllChildren() DeleterOption {
	return func(dc *deleterImpl) {
		dc.gCFunc = func([]byte) bool { return true }
	}
}

// GCMatchingChildren makes the deleter delete all children referenced from a deleted key that match the input function.
func GCMatchingChildren(kmf KeyMatchFunction) DeleterOption {
	return func(dc *deleterImpl) {
		if dc.gCFunc == nil {
			dc.gCFunc = kmf
		} else {
			dc.gCFunc = or(dc.gCFunc, kmf)
		}
	}
}

func or(kmf1, kmf2 KeyMatchFunction) KeyMatchFunction {
	return func(key []byte) bool {
		return kmf1(key) || kmf2(key)
	}
}
