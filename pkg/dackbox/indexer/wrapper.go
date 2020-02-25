package indexer

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/badgerhelper"
)

// Wrapper is an object that wraps keys and values into their indexed id:value pair.
//go:generate mockgen-wrapper
type Wrapper interface {
	Wrap(key []byte, msg proto.Message) (string, interface{})
}

// WrapperConfig is the configuration for a wrapper to add things to the index.
type WrapperConfig struct {
	prefix  []byte
	wrapper Wrapper
}

// NewWrapper returns a new wrapper which applies the input wrappers when their corresponding prefix is seen.
func NewWrapper(configs ...WrapperConfig) Wrapper {
	wrappers := make(map[string]Wrapper, len(configs))
	for _, config := range configs {
		wrappers[string(config.prefix)] = config.wrapper
	}
	return &wrapperImpl{wrappers: wrappers}
}

type wrapperImpl struct {
	wrappers map[string]Wrapper
}

func (w *wrapperImpl) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	longestMatch := findLongestMatch(w.wrappers, key)
	if longestMatch != nil {
		return longestMatch.Wrap(key, msg)
	}
	return "", nil
}

func findLongestMatch(wrappers map[string]Wrapper, key []byte) Wrapper {
	// Need to find the longest matching prefix for a registered index.
	var totalPrefix []byte
	var longestMatch Wrapper
	for currPrefix := badgerhelper.GetPrefix(key); len(currPrefix) > 0; currPrefix = badgerhelper.GetPrefix(badgerhelper.StripPrefix(totalPrefix, key)) {
		totalPrefix = append(totalPrefix, currPrefix...)
		if match, contains := wrappers[string(totalPrefix)]; contains {
			longestMatch = match
		}
	}
	return longestMatch
}
