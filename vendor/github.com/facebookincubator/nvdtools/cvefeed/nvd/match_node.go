// Copyright (c) Facebook, Inc. and its affiliates.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nvd

import (
	"fmt"
	"log"
	"strings"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/facebookincubator/nvdtools/wfn"
)

// Matcher returns an object which knows how to match attributes
func nodeMatcher(node *schema.NVDCVEFeedJSON10DefNode) (wfn.Matcher, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	var ms []wfn.Matcher
	for _, match := range node.CPEMatch {
		if match != nil {
			if m, err := cpeMatcher(match); err == nil {
				ms = append(ms, m)
			}
		}
	}
	for _, child := range node.Children {
		if child != nil {
			if m, err := nodeMatcher(child); err == nil {
				ms = append(ms, m)
			}
		}
	}

	if len(ms) == 0 {
		return nil, fmt.Errorf("empty configuration for node")
	}

	var m wfn.Matcher

	switch strings.ToUpper(node.Operator) {
	default:
		log.Printf("unknown operator, defaulting to OR: got %q", node.Operator)
		fallthrough
	case "OR":
		m = wfn.MatchAny(ms...)
	case "AND":
		m = wfn.MatchAll(ms...)
	}

	if node.Negate {
		m = wfn.DontMatch(m)
	}

	return m, nil
}
