// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: tools/local-sensor/message.proto

package localSensor

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	storage "github.com/stackrox/rox/generated/storage"
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

type LocalSensorPolicies struct {
	Policies             []*storage.Policy `protobuf:"bytes,1,rep,name=policies,proto3" json:"policies,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *LocalSensorPolicies) Reset()         { *m = LocalSensorPolicies{} }
func (m *LocalSensorPolicies) String() string { return proto.CompactTextString(m) }
func (*LocalSensorPolicies) ProtoMessage()    {}
func (*LocalSensorPolicies) Descriptor() ([]byte, []int) {
	return fileDescriptor_16761c5ea3753077, []int{0}
}
func (m *LocalSensorPolicies) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *LocalSensorPolicies) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_LocalSensorPolicies.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *LocalSensorPolicies) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LocalSensorPolicies.Merge(m, src)
}
func (m *LocalSensorPolicies) XXX_Size() int {
	return m.Size()
}
func (m *LocalSensorPolicies) XXX_DiscardUnknown() {
	xxx_messageInfo_LocalSensorPolicies.DiscardUnknown(m)
}

var xxx_messageInfo_LocalSensorPolicies proto.InternalMessageInfo

func (m *LocalSensorPolicies) GetPolicies() []*storage.Policy {
	if m != nil {
		return m.Policies
	}
	return nil
}

func (m *LocalSensorPolicies) MessageClone() proto.Message {
	return m.Clone()
}
func (m *LocalSensorPolicies) Clone() *LocalSensorPolicies {
	if m == nil {
		return nil
	}
	cloned := new(LocalSensorPolicies)
	*cloned = *m

	if m.Policies != nil {
		cloned.Policies = make([]*storage.Policy, len(m.Policies))
		for idx, v := range m.Policies {
			cloned.Policies[idx] = v.Clone()
		}
	}
	return cloned
}

func init() {
	proto.RegisterType((*LocalSensorPolicies)(nil), "localSensor.LocalSensorPolicies")
}

func init() { proto.RegisterFile("tools/local-sensor/message.proto", fileDescriptor_16761c5ea3753077) }

var fileDescriptor_16761c5ea3753077 = []byte{
	// 145 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x52, 0x28, 0xc9, 0xcf, 0xcf,
	0x29, 0xd6, 0xcf, 0xc9, 0x4f, 0x4e, 0xcc, 0xd1, 0x2d, 0x4e, 0xcd, 0x2b, 0xce, 0x2f, 0xd2, 0xcf,
	0x4d, 0x2d, 0x2e, 0x4e, 0x4c, 0x4f, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0xe2, 0x06, 0xcb,
	0x05, 0x83, 0xa5, 0xa4, 0x44, 0x8a, 0x4b, 0xf2, 0x8b, 0x12, 0xd3, 0x53, 0xf5, 0x0b, 0xf2, 0x73,
	0x32, 0x93, 0x2b, 0x21, 0x4a, 0x94, 0x9c, 0xb8, 0x84, 0x7d, 0x10, 0x8a, 0x02, 0x40, 0x52, 0x99,
	0xa9, 0xc5, 0x42, 0xda, 0x5c, 0x1c, 0x05, 0x50, 0xb6, 0x04, 0xa3, 0x02, 0xb3, 0x06, 0xb7, 0x11,
	0xbf, 0x1e, 0x54, 0xbf, 0x1e, 0x58, 0x51, 0x65, 0x10, 0x5c, 0x81, 0x93, 0xc0, 0x89, 0x47, 0x72,
	0x8c, 0x17, 0x1e, 0xc9, 0x31, 0x3e, 0x78, 0x24, 0xc7, 0x38, 0xe3, 0xb1, 0x1c, 0x43, 0x12, 0x1b,
	0xd8, 0x70, 0x63, 0x40, 0x00, 0x00, 0x00, 0xff, 0xff, 0xfc, 0x41, 0xc0, 0x30, 0xa3, 0x00, 0x00,
	0x00,
}

func (m *LocalSensorPolicies) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *LocalSensorPolicies) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *LocalSensorPolicies) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Policies) > 0 {
		for iNdEx := len(m.Policies) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Policies[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintMessage(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0xa
		}
	}
	return len(dAtA) - i, nil
}

func encodeVarintMessage(dAtA []byte, offset int, v uint64) int {
	offset -= sovMessage(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *LocalSensorPolicies) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if len(m.Policies) > 0 {
		for _, e := range m.Policies {
			l = e.Size()
			n += 1 + l + sovMessage(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovMessage(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozMessage(x uint64) (n int) {
	return sovMessage(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *LocalSensorPolicies) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowMessage
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
			return fmt.Errorf("proto: LocalSensorPolicies: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: LocalSensorPolicies: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Policies", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowMessage
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
				return ErrInvalidLengthMessage
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthMessage
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Policies = append(m.Policies, &storage.Policy{})
			if err := m.Policies[len(m.Policies)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipMessage(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthMessage
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
func skipMessage(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowMessage
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
					return 0, ErrIntOverflowMessage
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
					return 0, ErrIntOverflowMessage
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
				return 0, ErrInvalidLengthMessage
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupMessage
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthMessage
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthMessage        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowMessage          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupMessage = fmt.Errorf("proto: unexpected end of group")
)
