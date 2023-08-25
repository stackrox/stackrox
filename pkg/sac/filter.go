package sac

// FilterSlice filters the given typed slice, applying a typed predicate function to obtain scope keys.
func FilterSlice[T any](sc ScopeChecker, objSlice []T, scopePredFunc func(T) ScopePredicate) []T {
	if sc.IsAllowed() {
		return objSlice
	}

	allowedObjs := make([]T, 0, len(objSlice))
	for _, obj := range objSlice {
		pred := scopePredFunc(obj)
		if pred.Allowed(sc) {
			allowedObjs = append(allowedObjs, obj)
		}
	}

	return allowedObjs
}

// FilterMap filters the given typed map, applying a typed predicate function to obtain scope keys. The arguments
// passed to the scope predicate function are the key and value of each map entry.
func FilterMap[K comparable, V any](sc ScopeChecker, objMap map[K]V, scopePredFunc func(K, V) ScopePredicate) map[K]V {
	if sc.IsAllowed() {
		return objMap
	}

	allowed := make(map[K]V, len(objMap))
	for k, v := range objMap {
		pred := scopePredFunc(k, v)
		if pred.Allowed(sc) {
			allowed[k] = v
		}
	}
	return allowed
}
