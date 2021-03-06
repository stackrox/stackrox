// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: api/v1/credential_expiry_service.proto

package v1

import (
	context "context"
	fmt "fmt"
	types "github.com/gogo/protobuf/types"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type GetCertExpiry_Component int32

const (
	GetCertExpiry_UNKNOWN GetCertExpiry_Component = 0
	GetCertExpiry_CENTRAL GetCertExpiry_Component = 1
	GetCertExpiry_SCANNER GetCertExpiry_Component = 2
)

var GetCertExpiry_Component_name = map[int32]string{
	0: "UNKNOWN",
	1: "CENTRAL",
	2: "SCANNER",
}

var GetCertExpiry_Component_value = map[string]int32{
	"UNKNOWN": 0,
	"CENTRAL": 1,
	"SCANNER": 2,
}

func (x GetCertExpiry_Component) String() string {
	return proto.EnumName(GetCertExpiry_Component_name, int32(x))
}

func (GetCertExpiry_Component) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_bd0d8e0eb298005f, []int{0, 0}
}

type GetCertExpiry struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetCertExpiry) Reset()         { *m = GetCertExpiry{} }
func (m *GetCertExpiry) String() string { return proto.CompactTextString(m) }
func (*GetCertExpiry) ProtoMessage()    {}
func (*GetCertExpiry) Descriptor() ([]byte, []int) {
	return fileDescriptor_bd0d8e0eb298005f, []int{0}
}
func (m *GetCertExpiry) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GetCertExpiry) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GetCertExpiry.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GetCertExpiry) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetCertExpiry.Merge(m, src)
}
func (m *GetCertExpiry) XXX_Size() int {
	return m.Size()
}
func (m *GetCertExpiry) XXX_DiscardUnknown() {
	xxx_messageInfo_GetCertExpiry.DiscardUnknown(m)
}

var xxx_messageInfo_GetCertExpiry proto.InternalMessageInfo

func (m *GetCertExpiry) MessageClone() proto.Message {
	return m.Clone()
}
func (m *GetCertExpiry) Clone() *GetCertExpiry {
	if m == nil {
		return nil
	}
	cloned := new(GetCertExpiry)
	*cloned = *m

	return cloned
}

type GetCertExpiry_Request struct {
	Component            GetCertExpiry_Component `protobuf:"varint,1,opt,name=component,proto3,enum=v1.GetCertExpiry_Component" json:"component,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *GetCertExpiry_Request) Reset()         { *m = GetCertExpiry_Request{} }
func (m *GetCertExpiry_Request) String() string { return proto.CompactTextString(m) }
func (*GetCertExpiry_Request) ProtoMessage()    {}
func (*GetCertExpiry_Request) Descriptor() ([]byte, []int) {
	return fileDescriptor_bd0d8e0eb298005f, []int{0, 0}
}
func (m *GetCertExpiry_Request) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GetCertExpiry_Request) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GetCertExpiry_Request.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GetCertExpiry_Request) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetCertExpiry_Request.Merge(m, src)
}
func (m *GetCertExpiry_Request) XXX_Size() int {
	return m.Size()
}
func (m *GetCertExpiry_Request) XXX_DiscardUnknown() {
	xxx_messageInfo_GetCertExpiry_Request.DiscardUnknown(m)
}

var xxx_messageInfo_GetCertExpiry_Request proto.InternalMessageInfo

func (m *GetCertExpiry_Request) GetComponent() GetCertExpiry_Component {
	if m != nil {
		return m.Component
	}
	return GetCertExpiry_UNKNOWN
}

func (m *GetCertExpiry_Request) MessageClone() proto.Message {
	return m.Clone()
}
func (m *GetCertExpiry_Request) Clone() *GetCertExpiry_Request {
	if m == nil {
		return nil
	}
	cloned := new(GetCertExpiry_Request)
	*cloned = *m

	return cloned
}

type GetCertExpiry_Response struct {
	Expiry               *types.Timestamp `protobuf:"bytes,1,opt,name=expiry,proto3" json:"expiry,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *GetCertExpiry_Response) Reset()         { *m = GetCertExpiry_Response{} }
func (m *GetCertExpiry_Response) String() string { return proto.CompactTextString(m) }
func (*GetCertExpiry_Response) ProtoMessage()    {}
func (*GetCertExpiry_Response) Descriptor() ([]byte, []int) {
	return fileDescriptor_bd0d8e0eb298005f, []int{0, 1}
}
func (m *GetCertExpiry_Response) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *GetCertExpiry_Response) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_GetCertExpiry_Response.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *GetCertExpiry_Response) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetCertExpiry_Response.Merge(m, src)
}
func (m *GetCertExpiry_Response) XXX_Size() int {
	return m.Size()
}
func (m *GetCertExpiry_Response) XXX_DiscardUnknown() {
	xxx_messageInfo_GetCertExpiry_Response.DiscardUnknown(m)
}

var xxx_messageInfo_GetCertExpiry_Response proto.InternalMessageInfo

func (m *GetCertExpiry_Response) GetExpiry() *types.Timestamp {
	if m != nil {
		return m.Expiry
	}
	return nil
}

func (m *GetCertExpiry_Response) MessageClone() proto.Message {
	return m.Clone()
}
func (m *GetCertExpiry_Response) Clone() *GetCertExpiry_Response {
	if m == nil {
		return nil
	}
	cloned := new(GetCertExpiry_Response)
	*cloned = *m

	cloned.Expiry = m.Expiry.Clone()
	return cloned
}

func init() {
	proto.RegisterEnum("v1.GetCertExpiry_Component", GetCertExpiry_Component_name, GetCertExpiry_Component_value)
	proto.RegisterType((*GetCertExpiry)(nil), "v1.GetCertExpiry")
	proto.RegisterType((*GetCertExpiry_Request)(nil), "v1.GetCertExpiry.Request")
	proto.RegisterType((*GetCertExpiry_Response)(nil), "v1.GetCertExpiry.Response")
}

func init() {
	proto.RegisterFile("api/v1/credential_expiry_service.proto", fileDescriptor_bd0d8e0eb298005f)
}

var fileDescriptor_bd0d8e0eb298005f = []byte{
	// 348 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x4b, 0x2c, 0xc8, 0xd4,
	0x2f, 0x33, 0xd4, 0x4f, 0x2e, 0x4a, 0x4d, 0x49, 0xcd, 0x2b, 0xc9, 0x4c, 0xcc, 0x89, 0x4f, 0xad,
	0x28, 0xc8, 0x2c, 0xaa, 0x8c, 0x2f, 0x4e, 0x2d, 0x2a, 0xcb, 0x4c, 0x4e, 0xd5, 0x2b, 0x28, 0xca,
	0x2f, 0xc9, 0x17, 0x62, 0x2a, 0x33, 0x94, 0x92, 0x49, 0xcf, 0xcf, 0x4f, 0xcf, 0x49, 0xd5, 0x07,
	0x69, 0x49, 0xcc, 0xcb, 0xcb, 0x2f, 0x49, 0x2c, 0xc9, 0xcc, 0xcf, 0x2b, 0x86, 0xa8, 0x90, 0x92,
	0x87, 0xca, 0x82, 0x79, 0x49, 0xa5, 0x69, 0xfa, 0x25, 0x99, 0xb9, 0xa9, 0xc5, 0x25, 0x89, 0xb9,
	0x05, 0x10, 0x05, 0x4a, 0x27, 0x19, 0xb9, 0x78, 0xdd, 0x53, 0x4b, 0x9c, 0x53, 0x8b, 0x4a, 0x5c,
	0xc1, 0x56, 0x48, 0xb9, 0x70, 0xb1, 0x07, 0xa5, 0x16, 0x96, 0xa6, 0x16, 0x97, 0x08, 0x59, 0x72,
	0x71, 0x26, 0xe7, 0xe7, 0x16, 0xe4, 0xe7, 0xa5, 0xe6, 0x95, 0x48, 0x30, 0x2a, 0x30, 0x6a, 0xf0,
	0x19, 0x49, 0xeb, 0x95, 0x19, 0xea, 0xa1, 0x68, 0xd0, 0x73, 0x86, 0x29, 0x09, 0x42, 0xa8, 0x96,
	0xb2, 0xe3, 0xe2, 0x08, 0x4a, 0x2d, 0x2e, 0xc8, 0xcf, 0x2b, 0x4e, 0x15, 0x32, 0xe2, 0x62, 0x83,
	0x38, 0x1f, 0x6c, 0x06, 0xb7, 0x91, 0x94, 0x1e, 0xc4, 0x55, 0x7a, 0x30, 0x57, 0xe9, 0x85, 0xc0,
	0x5c, 0x15, 0x04, 0x55, 0xa9, 0x64, 0xc4, 0xc5, 0x09, 0x37, 0x57, 0x88, 0x9b, 0x8b, 0x3d, 0xd4,
	0xcf, 0xdb, 0xcf, 0x3f, 0xdc, 0x4f, 0x80, 0x01, 0xc4, 0x71, 0x76, 0xf5, 0x0b, 0x09, 0x72, 0xf4,
	0x11, 0x60, 0x04, 0x71, 0x82, 0x9d, 0x1d, 0xfd, 0xfc, 0x5c, 0x83, 0x04, 0x98, 0x8c, 0xea, 0xb9,
	0xc4, 0x9d, 0xe1, 0x21, 0x06, 0x71, 0x5c, 0x30, 0x24, 0xbc, 0x84, 0x52, 0xd0, 0x7c, 0x29, 0x24,
	0x89, 0xe9, 0x0f, 0xa8, 0xaf, 0xa5, 0xa4, 0xb0, 0x49, 0x41, 0xbc, 0xa2, 0x24, 0xd3, 0x74, 0xf9,
	0xc9, 0x64, 0x26, 0x31, 0x21, 0x11, 0xd4, 0xe8, 0x81, 0x38, 0xda, 0x49, 0xef, 0xc4, 0x23, 0x39,
	0xc6, 0x0b, 0x8f, 0xe4, 0x18, 0x1f, 0x3c, 0x92, 0x63, 0x9c, 0xf1, 0x58, 0x8e, 0x81, 0x4b, 0x22,
	0x33, 0x5f, 0xaf, 0xb8, 0x24, 0x31, 0x39, 0xbb, 0x28, 0xbf, 0x02, 0xe2, 0x5d, 0xbd, 0xc4, 0x82,
	0x4c, 0xbd, 0x32, 0xc3, 0x28, 0xa6, 0x32, 0xc3, 0x08, 0x86, 0x24, 0x36, 0xb0, 0x98, 0x31, 0x20,
	0x00, 0x00, 0xff, 0xff, 0x51, 0x38, 0x87, 0x1f, 0xf2, 0x01, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// CredentialExpiryServiceClient is the client API for CredentialExpiryService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConnInterface.NewStream.
type CredentialExpiryServiceClient interface {
	// GetCertExpiry returns information related to the expiry component mTLS certificate.
	GetCertExpiry(ctx context.Context, in *GetCertExpiry_Request, opts ...grpc.CallOption) (*GetCertExpiry_Response, error)
}

type credentialExpiryServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCredentialExpiryServiceClient(cc grpc.ClientConnInterface) CredentialExpiryServiceClient {
	return &credentialExpiryServiceClient{cc}
}

func (c *credentialExpiryServiceClient) GetCertExpiry(ctx context.Context, in *GetCertExpiry_Request, opts ...grpc.CallOption) (*GetCertExpiry_Response, error) {
	out := new(GetCertExpiry_Response)
	err := c.cc.Invoke(ctx, "/v1.CredentialExpiryService/GetCertExpiry", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CredentialExpiryServiceServer is the server API for CredentialExpiryService service.
type CredentialExpiryServiceServer interface {
	// GetCertExpiry returns information related to the expiry component mTLS certificate.
	GetCertExpiry(context.Context, *GetCertExpiry_Request) (*GetCertExpiry_Response, error)
}

// UnimplementedCredentialExpiryServiceServer can be embedded to have forward compatible implementations.
type UnimplementedCredentialExpiryServiceServer struct {
}

func (*UnimplementedCredentialExpiryServiceServer) GetCertExpiry(ctx context.Context, req *GetCertExpiry_Request) (*GetCertExpiry_Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetCertExpiry not implemented")
}

func RegisterCredentialExpiryServiceServer(s *grpc.Server, srv CredentialExpiryServiceServer) {
	s.RegisterService(&_CredentialExpiryService_serviceDesc, srv)
}

func _CredentialExpiryService_GetCertExpiry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetCertExpiry_Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CredentialExpiryServiceServer).GetCertExpiry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/v1.CredentialExpiryService/GetCertExpiry",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CredentialExpiryServiceServer).GetCertExpiry(ctx, req.(*GetCertExpiry_Request))
	}
	return interceptor(ctx, in, info, handler)
}

var _CredentialExpiryService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "v1.CredentialExpiryService",
	HandlerType: (*CredentialExpiryServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetCertExpiry",
			Handler:    _CredentialExpiryService_GetCertExpiry_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/v1/credential_expiry_service.proto",
}

func (m *GetCertExpiry) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetCertExpiry) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GetCertExpiry) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	return len(dAtA) - i, nil
}

func (m *GetCertExpiry_Request) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetCertExpiry_Request) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GetCertExpiry_Request) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Component != 0 {
		i = encodeVarintCredentialExpiryService(dAtA, i, uint64(m.Component))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *GetCertExpiry_Response) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *GetCertExpiry_Response) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *GetCertExpiry_Response) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Expiry != nil {
		{
			size, err := m.Expiry.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintCredentialExpiryService(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintCredentialExpiryService(dAtA []byte, offset int, v uint64) int {
	offset -= sovCredentialExpiryService(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *GetCertExpiry) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *GetCertExpiry_Request) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Component != 0 {
		n += 1 + sovCredentialExpiryService(uint64(m.Component))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *GetCertExpiry_Response) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Expiry != nil {
		l = m.Expiry.Size()
		n += 1 + l + sovCredentialExpiryService(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovCredentialExpiryService(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozCredentialExpiryService(x uint64) (n int) {
	return sovCredentialExpiryService(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *GetCertExpiry) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCredentialExpiryService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: GetCertExpiry: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: GetCertExpiry: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipCredentialExpiryService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCredentialExpiryService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *GetCertExpiry_Request) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCredentialExpiryService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Request: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Request: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Component", wireType)
			}
			m.Component = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCredentialExpiryService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Component |= GetCertExpiry_Component(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipCredentialExpiryService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCredentialExpiryService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *GetCertExpiry_Response) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowCredentialExpiryService
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Response: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Response: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Expiry", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowCredentialExpiryService
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthCredentialExpiryService
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthCredentialExpiryService
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Expiry == nil {
				m.Expiry = &types.Timestamp{}
			}
			if err := m.Expiry.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipCredentialExpiryService(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthCredentialExpiryService
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipCredentialExpiryService(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowCredentialExpiryService
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCredentialExpiryService
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowCredentialExpiryService
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthCredentialExpiryService
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupCredentialExpiryService
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthCredentialExpiryService
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthCredentialExpiryService        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowCredentialExpiryService          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupCredentialExpiryService = fmt.Errorf("proto: unexpected end of group")
)
