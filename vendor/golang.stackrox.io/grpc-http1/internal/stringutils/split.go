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

package stringutils

import "strings"

// Split2 splits the given string at the given separator, returning the part before and after the separator as two
// separate return values.
// If the string does not contain `sep`, the entire string is returned as the first return value.
func Split2(str string, sep string) (string, string) {
	splitIdx := strings.Index(str, sep)
	if splitIdx == -1 {
		return str, ""
	}
	return str[:splitIdx], str[splitIdx+len(sep):]
}
