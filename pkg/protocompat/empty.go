package protocompat

import (
	"google.golang.org/protobuf/types/known/emptypb"
)

// Empty is a generic empty message that you can re-use to avoid defining duplicated empty messages
// in your APIs. A typical example is to use it as the request or the response type of an API method.
// For instance:
//
//	service Foo {
//	  rpc Bar(google.protobuf.Empty) returns (google.protobuf.Empty);
//	}
//
// The JSON representation for `Empty` is empty JSON object `{}`.
type Empty = emptypb.Empty

// ProtoEmpty returns a pointer to an instance of the generic empty type.
func ProtoEmpty() *Empty { return &Empty{} }
