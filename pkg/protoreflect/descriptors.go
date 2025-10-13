package protoreflect

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/protobuf/proto"
	descriptor "google.golang.org/protobuf/types/descriptorpb"
)

var (
	fileDescCache      = make(map[sliceIdentity]*descriptor.FileDescriptorProto)
	fileDescCacheMutex sync.RWMutex
)

// ProtoEnum is an interface implemented by all protobuf enums.
type ProtoEnum interface {
	EnumDescriptor() ([]byte, []int)
}

// parseFileDescriptor takes a gzipped serialized file descriptor proto, and returns the parsed proto object or an
// error.
func parseFileDescriptor(data []byte) (*descriptor.FileDescriptorProto, error) {
	dataSliceID := identityOfSlice(data)

	desc := concurrency.WithRLock1(&fileDescCacheMutex, func() *descriptor.FileDescriptorProto {
		d := fileDescCache[dataSliceID]
		return d
	})

	if desc != nil {
		return desc, nil
	}

	uncompressedReader, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, errors.Wrap(err, "uncompressing file descriptor data")
	}
	uncompressedData, err := io.ReadAll(uncompressedReader)
	if err != nil {
		return nil, errors.Wrap(err, "uncompressing file descriptor data")
	}
	desc = &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(uncompressedData, desc); err != nil {
		return nil, errors.Wrap(err, "unmarshalling file descriptor")
	}

	concurrency.WithLock(&fileDescCacheMutex, func() {
		fileDescCache[dataSliceID] = desc
	})

	return desc, nil
}

// scope is an interface for unifying descriptors that may have nested types (i.e., FileDescriptorProto and
// DescriptorProto).
type scope interface {
	GetName() string
	GetNestedType() []*descriptor.DescriptorProto
	GetEnumType() []*descriptor.EnumDescriptorProto
}

// fileDescWrap wraps a FileDescriptorProto to make it conform to the `scope` interface.
type fileDescWrap struct {
	*descriptor.FileDescriptorProto
}

func (d fileDescWrap) GetNestedType() []*descriptor.DescriptorProto {
	return d.GetMessageType()
}

// traverse resolves the transitively nested scope referred to by path.
func traverse(start scope, path []int) (scope, error) {
	curr := start
	for _, elem := range path {
		nested := curr.GetNestedType()
		if elem < 0 || elem >= len(nested) {
			return nil, fmt.Errorf("nested type index %d out of range in scope %s", elem, curr.GetName())
		}
		curr = nested[elem]
	}
	return curr, nil
}

// GetEnumDescriptor returns the EnumDescriptorProto for an enum type.
func GetEnumDescriptor(e ProtoEnum) (*descriptor.EnumDescriptorProto, error) {
	fileDescData, path := e.EnumDescriptor()
	fileDesc, err := parseFileDescriptor(fileDescData)
	if err != nil {
		return nil, errors.Wrap(err, "parsing enum descriptor")
	}
	inner, err := traverse(fileDescWrap{FileDescriptorProto: fileDesc}, path[:len(path)-1])
	if err != nil {
		return nil, errors.Wrap(err, "resolving path to enum")
	}
	enumIdx := path[len(path)-1]
	enums := inner.GetEnumType()
	if enumIdx < 0 || enumIdx >= len(enums) {
		return nil, fmt.Errorf("invalid enum index %d in scope %s", enumIdx, inner.GetName())
	}
	return enums[enumIdx], nil
}

// ProtoMessage is an interface implemented by all protobuf messages.
type ProtoMessage interface {
	Descriptor() ([]byte, []int)
}

// GetMessageDescriptor returns the DescriptorProto for a protobuf object, or an error if the descriptor could not
// be determined.
func GetMessageDescriptor(pb ProtoMessage) (*descriptor.DescriptorProto, error) {
	fileDescData, path := pb.Descriptor()
	fileDesc, err := parseFileDescriptor(fileDescData)
	if err != nil {
		return nil, errors.Wrap(err, "parsing message descriptor")
	}
	innermost, err := traverse(fileDescWrap{FileDescriptorProto: fileDesc}, path)
	if err != nil {
		return nil, errors.Wrap(err, "resolving path to message")
	}
	messageDesc, ok := innermost.(*descriptor.DescriptorProto)
	if !ok {
		return nil, fmt.Errorf("innermost scope is not a DescriptorProto but %T", innermost)
	}
	return messageDesc, nil
}
