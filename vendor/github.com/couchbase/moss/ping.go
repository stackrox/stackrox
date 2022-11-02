//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

// A ping message is used to notify and wait for asynchronous tasks.
type ping struct {
	kind string // The kind of ping.

	// When non-nil, the pongCh will be closed when task is done.
	pongCh chan struct{}
}

// replyToPings() is a helper function to respond to ping requests.
func replyToPings(pings []ping) {
	for _, ping := range pings {
		if ping.pongCh != nil {
			close(ping.pongCh)
			ping.pongCh = nil
		}
	}
}

// receivePings() collects any available ping requests, but will not
// block if there are no incoming ping requests.
func receivePings(pingCh chan ping, pings []ping,
	kindMatch string, kindSeen bool) ([]ping, bool) {
	for {
		select {
		case pingVal := <-pingCh:
			pings = append(pings, pingVal)
			if pingVal.kind == kindMatch {
				kindSeen = true
			}

		default:
			return pings, kindSeen
		}
	}
}
