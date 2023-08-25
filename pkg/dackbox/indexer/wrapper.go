package indexer

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/dbhelper"
)

// Wrapper is an object that wraps keys and values into their indexed id:value pair.
//
//go:generate mockgen-wrapper
type Wrapper interface {
	Wrap(key []byte, msg proto.Message) (string, interface{})
}

func findLongestMatch(wrappers map[string]Wrapper, key []byte) Wrapper {
	// Need to find the longest matching prefix for a registered index.
	var totalPrefix []byte
	var longestMatch Wrapper
	for currPrefix := dbhelper.GetPrefix(key); len(currPrefix) > 0; currPrefix = dbhelper.GetPrefix(dbhelper.StripPrefix(totalPrefix, key)) {
		totalPrefix = append(totalPrefix, currPrefix...)
		if match, contains := wrappers[string(totalPrefix)]; contains {
			longestMatch = match
		}
	}
	return longestMatch
}
