// Copyright (c) 2020 StackRox Inc.
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
// limitations under the License

package concurrency

import (
	"time"
)

// Do performs the action as soon as the waitable is done.
// It blocks indefinitely until that happens.
func Do(w Waitable, action func()) {
	Wait(w)
	action()
}

// DoWithTimeout performs the action as soon as the waitable is done.
// It gives up and returns after timeout, and returns a bool indicating whether
// the action was performed or not.
func DoWithTimeout(w Waitable, action func(), timeout time.Duration) bool {
	if WaitWithTimeout(w, timeout) {
		action()
		return true
	}
	return false
}
