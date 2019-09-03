package common

import "k8s.io/apimachinery/pkg/runtime/schema"

var (
	// StateResourceTypes enumerates all resource types that are (or were at some point) used by the upgrader to manage
	// its own state.
	// IMPORTANT: Never remove from this list, otherwise upgrader state might linger around forever.
	StateResourceTypes = []schema.GroupVersionKind{
		{Version: "v1", Kind: "Secret"},
	}
)
