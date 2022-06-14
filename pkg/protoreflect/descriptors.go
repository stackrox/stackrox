package protoreflect

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/sync"
	"google.golang.org/grpc"
)

// FileDescLookupFn is a function that resolves the raw data of a gzipped serialized FileDescriptorProto for a given
// file name.
type FileDescLookupFn func(string) []byte

var (
	fileDescCache      = make(map[sliceIdentity]*descriptor.FileDescriptorProto)
	fileDescCacheMutex sync.RWMutex

	fileDescLookupFns          []FileDescLookupFn
	fileDescLookupFnIdentities map[uintptr]struct{}
	fileDescLookupFnsMutex     sync.RWMutex
)

// RegisterFileDescriptorLookup registers a function for looking up file descriptor data. Its primary purpose is to
// allow resolving file descriptor for alternative protobuf implementations (i.e., gogo protobuf) without introducing
// an explicit dependency.
func RegisterFileDescriptorLookup(fn FileDescLookupFn) {
	addr := reflect.ValueOf(fn).Pointer()
	if addr == reflect.ValueOf(proto.FileDescriptor).Pointer() {
		return // no need to register `proto.FileDescriptor`.
	}
	fileDescLookupFnsMutex.Lock()
	defer fileDescLookupFnsMutex.Unlock()

	if _, ok := fileDescLookupFnIdentities[addr]; ok {
		return
	}

	fileDescLookupFns = append(fileDescLookupFns, fn)
	fileDescLookupFnIdentities[addr] = struct{}{}
}

// ProtoEnum is an interface implemented by all protobuf enums.
type ProtoEnum interface {
	EnumDescriptor() ([]byte, []int)
}

// ParseFileDescriptor takes a gzipped serialized file descriptor proto, and returns the parsed proto object or an
// error.
func ParseFileDescriptor(data []byte) (*descriptor.FileDescriptorProto, error) {
	dataSliceID := identityOfSlice(data)

	fileDescCacheMutex.RLock()
	desc := fileDescCache[dataSliceID]
	fileDescCacheMutex.RUnlock()

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

	fileDescCacheMutex.Lock()
	fileDescCache[dataSliceID] = desc
	fileDescCacheMutex.Unlock()

	return desc, nil
}

// LookupFileDescriptorData attempts to retrieve the data (gzipped proto) of a FileDescriptorProto. In addition to the
// official protobuf library, it will consult any lookup functions registered via RegisterFileDescriptorLookup.
func LookupFileDescriptorData(fileName string) []byte {
	data := proto.FileDescriptor(fileName)
	if data != nil {
		return data
	}

	fileDescLookupFnsMutex.RLock()
	defer fileDescLookupFnsMutex.RUnlock()

	for _, fn := range fileDescLookupFns {
		data = fn(fileName)
		if data != nil {
			break
		}
	}
	return data
}

// GetFileDescriptor returns the file descriptor proto for the given file name, or an error if the file descriptor was
// not found.
func GetFileDescriptor(fileName string) (*descriptor.FileDescriptorProto, error) {
	data := LookupFileDescriptorData(fileName)
	if data == nil {
		return nil, fmt.Errorf("no descriptor registered for %s", fileName)
	}
	return ParseFileDescriptor(data)
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
	fileDesc, err := ParseFileDescriptor(fileDescData)
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
	fileDesc, err := ParseFileDescriptor(fileDescData)
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

// GetServiceDescriptor returns the ServiceDescriptorProto for a given service, or an error if the descriptor
// could not be retrieved.
func GetServiceDescriptor(serviceName string, info grpc.ServiceInfo) (*descriptor.ServiceDescriptorProto, error) {
	if info.Metadata == nil {
		return nil, fmt.Errorf("service info for %s has no metadata", serviceName)
	}

	switch md := info.Metadata.(type) {
	case string:
		fileDesc, err := GetFileDescriptor(md)
		if err != nil {
			return nil, err
		}
		serviceBaseName := serviceName
		if dotIdx := strings.LastIndex(serviceName, "."); dotIdx != -1 {
			serviceBaseName = serviceName[dotIdx+1:]
		}
		for _, serviceDesc := range fileDesc.GetService() {
			if serviceDesc.GetName() == serviceBaseName {
				return serviceDesc, nil
			}
		}

		return nil, fmt.Errorf("service %s not found in descriptor for %s", serviceBaseName, md)

	default:
		return nil, fmt.Errorf("unsupported metadata type %T for service %s", md, serviceName)
	}
}
