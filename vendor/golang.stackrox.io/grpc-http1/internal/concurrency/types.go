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

// Waitable is a generic interface for things that can be waited upon. The method `Done` returns a channel that, when
// closed, signals that whatever condition is represented by this waitable is satisfied.
// Note: The name `Done` was chosen such that `context.Context` conforms to this interface.
type Waitable interface {
	Done() <-chan struct{}
}

// WaitableChan is an alias around a `<-chan struct{}` that returns itself in its `Done` method.
type WaitableChan <-chan struct{}

// Done returns the channel itself.
func (c WaitableChan) Done() <-chan struct{} {
	return c
}
