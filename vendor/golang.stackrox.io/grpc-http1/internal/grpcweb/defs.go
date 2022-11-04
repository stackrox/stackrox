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

package grpcweb

const (
	compressedFlag     byte = 1 << 0
	trailerMessageFlag byte = 1 << 7

	completeHeaderLen = 5

	// GRPCWebOnlyHeader is a header to indicate that the server should always return gRPC-Web
	// responses, regardless of detected client capabilities. The presence of the header alone
	// is sufficient, however it is recommended that a client chooses "true" as the only value
	// whenver the header is used.
	GRPCWebOnlyHeader = `Grpc-Web-Only`
)
