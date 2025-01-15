// protoc-gen-go-vtproto version: v0.6.1-0.20240409071808-615f978279ca
// source: storage/image_component.proto

package storage

import (
	binary "encoding/binary"
	fmt "fmt"
	protohelpers "github.com/planetscale/vtprotobuf/protohelpers"
	proto "google.golang.org/protobuf/proto"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	io "io"
	math "math"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

func (m *ImageComponent) CloneVT() *ImageComponent {
	if m == nil {
		return (*ImageComponent)(nil)
	}
	r := new(ImageComponent)
	r.Id = m.Id
	r.Name = m.Name
	r.Version = m.Version
	r.License = m.License.CloneVT()
	r.Priority = m.Priority
	r.Source = m.Source
	r.RiskScore = m.RiskScore
	r.FixedBy = m.FixedBy
	r.OperatingSystem = m.OperatingSystem
	if m.SetTopCvss != nil {
		r.SetTopCvss = m.SetTopCvss.(interface {
			CloneVT() isImageComponent_SetTopCvss
		}).CloneVT()
	}
	if len(m.unknownFields) > 0 {
		r.unknownFields = make([]byte, len(m.unknownFields))
		copy(r.unknownFields, m.unknownFields)
	}
	return r
}

func (m *ImageComponent) CloneMessageVT() proto.Message {
	return m.CloneVT()
}

func (m *ImageComponent_TopCvss) CloneVT() isImageComponent_SetTopCvss {
	if m == nil {
		return (*ImageComponent_TopCvss)(nil)
	}
	r := new(ImageComponent_TopCvss)
	r.TopCvss = m.TopCvss
	return r
}

func (m *ImageComponentV2) CloneVT() *ImageComponentV2 {
	if m == nil {
		return (*ImageComponentV2)(nil)
	}
	r := new(ImageComponentV2)
	r.Id = m.Id
	r.Name = m.Name
	r.Version = m.Version
	r.License = m.License.CloneVT()
	r.Priority = m.Priority
	r.Source = m.Source
	r.RiskScore = m.RiskScore
	r.FixedBy = m.FixedBy
	r.OperatingSystem = m.OperatingSystem
	r.ImageId = m.ImageId
	r.Location = m.Location
	r.Architecture = m.Architecture
	if m.SetTopCvss != nil {
		r.SetTopCvss = m.SetTopCvss.(interface {
			CloneVT() isImageComponentV2_SetTopCvss
		}).CloneVT()
	}
	if m.HasLayerIndex != nil {
		r.HasLayerIndex = m.HasLayerIndex.(interface {
			CloneVT() isImageComponentV2_HasLayerIndex
		}).CloneVT()
	}
<<<<<<< HEAD
=======
	if rhs := m.Cves; rhs != nil {
		tmpContainer := make([]*ImageCVEV2, len(rhs))
		for k, v := range rhs {
			tmpContainer[k] = v.CloneVT()
		}
		r.Cves = tmpContainer
	}
>>>>>>> 672686754e (X-Smart-Squash: Squashed 5 commits:)
	if len(m.unknownFields) > 0 {
		r.unknownFields = make([]byte, len(m.unknownFields))
		copy(r.unknownFields, m.unknownFields)
	}
	return r
}

func (m *ImageComponentV2) CloneMessageVT() proto.Message {
	return m.CloneVT()
}

func (m *ImageComponentV2_TopCvss) CloneVT() isImageComponentV2_SetTopCvss {
	if m == nil {
		return (*ImageComponentV2_TopCvss)(nil)
	}
	r := new(ImageComponentV2_TopCvss)
	r.TopCvss = m.TopCvss
	return r
}

func (m *ImageComponentV2_LayerIndex) CloneVT() isImageComponentV2_HasLayerIndex {
	if m == nil {
		return (*ImageComponentV2_LayerIndex)(nil)
	}
	r := new(ImageComponentV2_LayerIndex)
	r.LayerIndex = m.LayerIndex
	return r
}

func (this *ImageComponent) EqualVT(that *ImageComponent) bool {
	if this == that {
		return true
	} else if this == nil || that == nil {
		return false
	}
	if this.SetTopCvss == nil && that.SetTopCvss != nil {
		return false
	} else if this.SetTopCvss != nil {
		if that.SetTopCvss == nil {
			return false
		}
		if !this.SetTopCvss.(interface {
			EqualVT(isImageComponent_SetTopCvss) bool
		}).EqualVT(that.SetTopCvss) {
			return false
		}
	}
	if this.Id != that.Id {
		return false
	}
	if this.Name != that.Name {
		return false
	}
	if this.Version != that.Version {
		return false
	}
	if !this.License.EqualVT(that.License) {
		return false
	}
	if this.Priority != that.Priority {
		return false
	}
	if this.Source != that.Source {
		return false
	}
	if this.RiskScore != that.RiskScore {
		return false
	}
	if this.FixedBy != that.FixedBy {
		return false
	}
	if this.OperatingSystem != that.OperatingSystem {
		return false
	}
	return string(this.unknownFields) == string(that.unknownFields)
}

func (this *ImageComponent) EqualMessageVT(thatMsg proto.Message) bool {
	that, ok := thatMsg.(*ImageComponent)
	if !ok {
		return false
	}
	return this.EqualVT(that)
}
func (this *ImageComponent_TopCvss) EqualVT(thatIface isImageComponent_SetTopCvss) bool {
	that, ok := thatIface.(*ImageComponent_TopCvss)
	if !ok {
		return false
	}
	if this == that {
		return true
	}
	if this == nil && that != nil || this != nil && that == nil {
		return false
	}
	if this.TopCvss != that.TopCvss {
		return false
	}
	return true
}

func (this *ImageComponentV2) EqualVT(that *ImageComponentV2) bool {
	if this == that {
		return true
	} else if this == nil || that == nil {
		return false
	}
	if this.SetTopCvss == nil && that.SetTopCvss != nil {
		return false
	} else if this.SetTopCvss != nil {
		if that.SetTopCvss == nil {
			return false
		}
		if !this.SetTopCvss.(interface {
			EqualVT(isImageComponentV2_SetTopCvss) bool
		}).EqualVT(that.SetTopCvss) {
			return false
		}
	}
	if this.HasLayerIndex == nil && that.HasLayerIndex != nil {
		return false
	} else if this.HasLayerIndex != nil {
		if that.HasLayerIndex == nil {
			return false
		}
		if !this.HasLayerIndex.(interface {
			EqualVT(isImageComponentV2_HasLayerIndex) bool
		}).EqualVT(that.HasLayerIndex) {
			return false
		}
	}
	if this.Id != that.Id {
		return false
	}
	if this.Name != that.Name {
		return false
	}
	if this.Version != that.Version {
		return false
	}
	if !this.License.EqualVT(that.License) {
		return false
	}
	if this.Priority != that.Priority {
		return false
	}
	if this.Source != that.Source {
		return false
	}
	if this.RiskScore != that.RiskScore {
		return false
	}
	if this.FixedBy != that.FixedBy {
		return false
	}
	if this.OperatingSystem != that.OperatingSystem {
		return false
	}
	if this.ImageId != that.ImageId {
		return false
	}
	if this.Location != that.Location {
		return false
	}
<<<<<<< HEAD
=======
	if len(this.Cves) != len(that.Cves) {
		return false
	}
	for i, vx := range this.Cves {
		vy := that.Cves[i]
		if p, q := vx, vy; p != q {
			if p == nil {
				p = &ImageCVEV2{}
			}
			if q == nil {
				q = &ImageCVEV2{}
			}
			if !p.EqualVT(q) {
				return false
			}
		}
	}
>>>>>>> 672686754e (X-Smart-Squash: Squashed 5 commits:)
	if this.Architecture != that.Architecture {
		return false
	}
	return string(this.unknownFields) == string(that.unknownFields)
}

func (this *ImageComponentV2) EqualMessageVT(thatMsg proto.Message) bool {
	that, ok := thatMsg.(*ImageComponentV2)
	if !ok {
		return false
	}
	return this.EqualVT(that)
}
func (this *ImageComponentV2_TopCvss) EqualVT(thatIface isImageComponentV2_SetTopCvss) bool {
	that, ok := thatIface.(*ImageComponentV2_TopCvss)
	if !ok {
		return false
	}
	if this == that {
		return true
	}
	if this == nil && that != nil || this != nil && that == nil {
		return false
	}
	if this.TopCvss != that.TopCvss {
		return false
	}
	return true
}

func (this *ImageComponentV2_LayerIndex) EqualVT(thatIface isImageComponentV2_HasLayerIndex) bool {
	that, ok := thatIface.(*ImageComponentV2_LayerIndex)
	if !ok {
		return false
	}
	if this == that {
		return true
	}
	if this == nil && that != nil || this != nil && that == nil {
		return false
	}
	if this.LayerIndex != that.LayerIndex {
		return false
	}
	return true
}

func (m *ImageComponent) MarshalVT() (dAtA []byte, err error) {
	if m == nil {
		return nil, nil
	}
	size := m.SizeVT()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBufferVT(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ImageComponent) MarshalToVT(dAtA []byte) (int, error) {
	size := m.SizeVT()
	return m.MarshalToSizedBufferVT(dAtA[:size])
}

func (m *ImageComponent) MarshalToSizedBufferVT(dAtA []byte) (int, error) {
	if m == nil {
		return 0, nil
	}
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.unknownFields != nil {
		i -= len(m.unknownFields)
		copy(dAtA[i:], m.unknownFields)
	}
	if vtmsg, ok := m.SetTopCvss.(interface {
		MarshalToSizedBufferVT([]byte) (int, error)
	}); ok {
		size, err := vtmsg.MarshalToSizedBufferVT(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
	}
	if len(m.OperatingSystem) > 0 {
		i -= len(m.OperatingSystem)
		copy(dAtA[i:], m.OperatingSystem)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.OperatingSystem)))
		i--
		dAtA[i] = 0x52
	}
	if len(m.FixedBy) > 0 {
		i -= len(m.FixedBy)
		copy(dAtA[i:], m.FixedBy)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.FixedBy)))
		i--
		dAtA[i] = 0x4a
	}
	if m.RiskScore != 0 {
		i -= 4
		binary.LittleEndian.PutUint32(dAtA[i:], uint32(math.Float32bits(float32(m.RiskScore))))
		i--
		dAtA[i] = 0x3d
	}
	if m.Source != 0 {
		i = protohelpers.EncodeVarint(dAtA, i, uint64(m.Source))
		i--
		dAtA[i] = 0x30
	}
	if m.Priority != 0 {
		i = protohelpers.EncodeVarint(dAtA, i, uint64(m.Priority))
		i--
		dAtA[i] = 0x28
	}
	if m.License != nil {
		size, err := m.License.MarshalToSizedBufferVT(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = protohelpers.EncodeVarint(dAtA, i, uint64(size))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Version) > 0 {
		i -= len(m.Version)
		copy(dAtA[i:], m.Version)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Version)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ImageComponent_TopCvss) MarshalToVT(dAtA []byte) (int, error) {
	size := m.SizeVT()
	return m.MarshalToSizedBufferVT(dAtA[:size])
}

func (m *ImageComponent_TopCvss) MarshalToSizedBufferVT(dAtA []byte) (int, error) {
	i := len(dAtA)
	i -= 4
	binary.LittleEndian.PutUint32(dAtA[i:], uint32(math.Float32bits(float32(m.TopCvss))))
	i--
	dAtA[i] = 0x45
	return len(dAtA) - i, nil
}
func (m *ImageComponentV2) MarshalVT() (dAtA []byte, err error) {
	if m == nil {
		return nil, nil
	}
	size := m.SizeVT()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBufferVT(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ImageComponentV2) MarshalToVT(dAtA []byte) (int, error) {
	size := m.SizeVT()
	return m.MarshalToSizedBufferVT(dAtA[:size])
}

func (m *ImageComponentV2) MarshalToSizedBufferVT(dAtA []byte) (int, error) {
	if m == nil {
		return 0, nil
	}
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.unknownFields != nil {
		i -= len(m.unknownFields)
		copy(dAtA[i:], m.unknownFields)
	}
	if vtmsg, ok := m.HasLayerIndex.(interface {
		MarshalToSizedBufferVT([]byte) (int, error)
	}); ok {
		size, err := vtmsg.MarshalToSizedBufferVT(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
	}
	if vtmsg, ok := m.SetTopCvss.(interface {
		MarshalToSizedBufferVT([]byte) (int, error)
	}); ok {
		size, err := vtmsg.MarshalToSizedBufferVT(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
	}
	if len(m.Architecture) > 0 {
		i -= len(m.Architecture)
		copy(dAtA[i:], m.Architecture)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Architecture)))
		i--
<<<<<<< HEAD
		dAtA[i] = 0x72
=======
		dAtA[i] = 0x7a
	}
	if len(m.Cves) > 0 {
		for iNdEx := len(m.Cves) - 1; iNdEx >= 0; iNdEx-- {
			size, err := m.Cves[iNdEx].MarshalToSizedBufferVT(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = protohelpers.EncodeVarint(dAtA, i, uint64(size))
			i--
			dAtA[i] = 0x72
		}
>>>>>>> 672686754e (X-Smart-Squash: Squashed 5 commits:)
	}
	if len(m.Location) > 0 {
		i -= len(m.Location)
		copy(dAtA[i:], m.Location)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Location)))
		i--
		dAtA[i] = 0x6a
	}
	if len(m.ImageId) > 0 {
		i -= len(m.ImageId)
		copy(dAtA[i:], m.ImageId)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.ImageId)))
		i--
		dAtA[i] = 0x5a
	}
	if len(m.OperatingSystem) > 0 {
		i -= len(m.OperatingSystem)
		copy(dAtA[i:], m.OperatingSystem)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.OperatingSystem)))
		i--
		dAtA[i] = 0x52
	}
	if len(m.FixedBy) > 0 {
		i -= len(m.FixedBy)
		copy(dAtA[i:], m.FixedBy)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.FixedBy)))
		i--
		dAtA[i] = 0x4a
	}
	if m.RiskScore != 0 {
		i -= 4
		binary.LittleEndian.PutUint32(dAtA[i:], uint32(math.Float32bits(float32(m.RiskScore))))
		i--
		dAtA[i] = 0x3d
	}
	if m.Source != 0 {
		i = protohelpers.EncodeVarint(dAtA, i, uint64(m.Source))
		i--
		dAtA[i] = 0x30
	}
	if m.Priority != 0 {
		i = protohelpers.EncodeVarint(dAtA, i, uint64(m.Priority))
		i--
		dAtA[i] = 0x28
	}
	if m.License != nil {
		size, err := m.License.MarshalToSizedBufferVT(dAtA[:i])
		if err != nil {
			return 0, err
		}
		i -= size
		i = protohelpers.EncodeVarint(dAtA, i, uint64(size))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Version) > 0 {
		i -= len(m.Version)
		copy(dAtA[i:], m.Version)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Version)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = protohelpers.EncodeVarint(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ImageComponentV2_TopCvss) MarshalToVT(dAtA []byte) (int, error) {
	size := m.SizeVT()
	return m.MarshalToSizedBufferVT(dAtA[:size])
}

func (m *ImageComponentV2_TopCvss) MarshalToSizedBufferVT(dAtA []byte) (int, error) {
	i := len(dAtA)
	i -= 4
	binary.LittleEndian.PutUint32(dAtA[i:], uint32(math.Float32bits(float32(m.TopCvss))))
	i--
	dAtA[i] = 0x45
	return len(dAtA) - i, nil
}
func (m *ImageComponentV2_LayerIndex) MarshalToVT(dAtA []byte) (int, error) {
	size := m.SizeVT()
	return m.MarshalToSizedBufferVT(dAtA[:size])
}

func (m *ImageComponentV2_LayerIndex) MarshalToSizedBufferVT(dAtA []byte) (int, error) {
	i := len(dAtA)
	i = protohelpers.EncodeVarint(dAtA, i, uint64(m.LayerIndex))
	i--
	dAtA[i] = 0x60
	return len(dAtA) - i, nil
}
func (m *ImageComponent) SizeVT() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.Version)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	if m.License != nil {
		l = m.License.SizeVT()
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	if m.Priority != 0 {
		n += 1 + protohelpers.SizeOfVarint(uint64(m.Priority))
	}
	if m.Source != 0 {
		n += 1 + protohelpers.SizeOfVarint(uint64(m.Source))
	}
	if m.RiskScore != 0 {
		n += 5
	}
	if vtmsg, ok := m.SetTopCvss.(interface{ SizeVT() int }); ok {
		n += vtmsg.SizeVT()
	}
	l = len(m.FixedBy)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.OperatingSystem)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	n += len(m.unknownFields)
	return n
}

func (m *ImageComponent_TopCvss) SizeVT() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 5
	return n
}
func (m *ImageComponentV2) SizeVT() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.Version)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	if m.License != nil {
		l = m.License.SizeVT()
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	if m.Priority != 0 {
		n += 1 + protohelpers.SizeOfVarint(uint64(m.Priority))
	}
	if m.Source != 0 {
		n += 1 + protohelpers.SizeOfVarint(uint64(m.Source))
	}
	if m.RiskScore != 0 {
		n += 5
	}
	if vtmsg, ok := m.SetTopCvss.(interface{ SizeVT() int }); ok {
		n += vtmsg.SizeVT()
	}
	l = len(m.FixedBy)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.OperatingSystem)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	l = len(m.ImageId)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	if vtmsg, ok := m.HasLayerIndex.(interface{ SizeVT() int }); ok {
		n += vtmsg.SizeVT()
	}
	l = len(m.Location)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
<<<<<<< HEAD
=======
	if len(m.Cves) > 0 {
		for _, e := range m.Cves {
			l = e.SizeVT()
			n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
		}
	}
>>>>>>> 672686754e (X-Smart-Squash: Squashed 5 commits:)
	l = len(m.Architecture)
	if l > 0 {
		n += 1 + l + protohelpers.SizeOfVarint(uint64(l))
	}
	n += len(m.unknownFields)
	return n
}

func (m *ImageComponentV2_TopCvss) SizeVT() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 5
	return n
}
func (m *ImageComponentV2_LayerIndex) SizeVT() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	n += 1 + protohelpers.SizeOfVarint(uint64(m.LayerIndex))
	return n
}
func (m *ImageComponent) UnmarshalVT(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return protohelpers.ErrIntOverflow
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
			return fmt.Errorf("proto: ImageComponent: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ImageComponent: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Version = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field License", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
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
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.License == nil {
				m.License = &License{}
			}
			if err := m.License.UnmarshalVT(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Priority", wireType)
			}
			m.Priority = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Priority |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Source", wireType)
			}
			m.Source = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Source |= SourceType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field RiskScore", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.RiskScore = float32(math.Float32frombits(v))
		case 8:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field TopCvss", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.SetTopCvss = &ImageComponent_TopCvss{TopCvss: float32(math.Float32frombits(v))}
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FixedBy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FixedBy = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OperatingSystem", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OperatingSystem = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := protohelpers.Skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return protohelpers.ErrInvalidLength
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.unknownFields = append(m.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ImageComponentV2) UnmarshalVT(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return protohelpers.ErrIntOverflow
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
			return fmt.Errorf("proto: ImageComponentV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ImageComponentV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Version = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field License", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
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
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.License == nil {
				m.License = &License{}
			}
			if err := m.License.UnmarshalVT(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Priority", wireType)
			}
			m.Priority = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Priority |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Source", wireType)
			}
			m.Source = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Source |= SourceType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field RiskScore", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.RiskScore = float32(math.Float32frombits(v))
		case 8:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field TopCvss", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.SetTopCvss = &ImageComponentV2_TopCvss{TopCvss: float32(math.Float32frombits(v))}
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FixedBy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FixedBy = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OperatingSystem", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.OperatingSystem = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ImageId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ImageId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 12:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LayerIndex", wireType)
			}
			var v int32
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.HasLayerIndex = &ImageComponentV2_LayerIndex{LayerIndex: v}
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Location", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Location = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Architecture", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Architecture = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := protohelpers.Skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return protohelpers.ErrInvalidLength
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.unknownFields = append(m.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ImageComponent) UnmarshalVTUnsafe(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return protohelpers.ErrIntOverflow
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
			return fmt.Errorf("proto: ImageComponent: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ImageComponent: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Id = stringValue
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Name = stringValue
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Version = stringValue
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field License", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
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
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.License == nil {
				m.License = &License{}
			}
			if err := m.License.UnmarshalVTUnsafe(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Priority", wireType)
			}
			m.Priority = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Priority |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Source", wireType)
			}
			m.Source = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Source |= SourceType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field RiskScore", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.RiskScore = float32(math.Float32frombits(v))
		case 8:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field TopCvss", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.SetTopCvss = &ImageComponent_TopCvss{TopCvss: float32(math.Float32frombits(v))}
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FixedBy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.FixedBy = stringValue
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OperatingSystem", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.OperatingSystem = stringValue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := protohelpers.Skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return protohelpers.ErrInvalidLength
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.unknownFields = append(m.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ImageComponentV2) UnmarshalVTUnsafe(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return protohelpers.ErrIntOverflow
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
			return fmt.Errorf("proto: ImageComponentV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ImageComponentV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Id = stringValue
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Name = stringValue
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Version = stringValue
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field License", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
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
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.License == nil {
				m.License = &License{}
			}
			if err := m.License.UnmarshalVTUnsafe(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Priority", wireType)
			}
			m.Priority = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Priority |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Source", wireType)
			}
			m.Source = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Source |= SourceType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field RiskScore", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.RiskScore = float32(math.Float32frombits(v))
		case 8:
			if wireType != 5 {
				return fmt.Errorf("proto: wrong wireType = %d for field TopCvss", wireType)
			}
			var v uint32
			if (iNdEx + 4) > l {
				return io.ErrUnexpectedEOF
			}
			v = uint32(binary.LittleEndian.Uint32(dAtA[iNdEx:]))
			iNdEx += 4
			m.SetTopCvss = &ImageComponentV2_TopCvss{TopCvss: float32(math.Float32frombits(v))}
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FixedBy", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.FixedBy = stringValue
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OperatingSystem", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.OperatingSystem = stringValue
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ImageId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.ImageId = stringValue
			iNdEx = postIndex
		case 12:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field LayerIndex", wireType)
			}
			var v int32
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.HasLayerIndex = &ImageComponentV2_LayerIndex{LayerIndex: v}
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Location", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Location = stringValue
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
<<<<<<< HEAD
=======
				return fmt.Errorf("proto: wrong wireType = %d for field Cves", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
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
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Cves = append(m.Cves, &ImageCVEV2{})
			if err := m.Cves[len(m.Cves)-1].UnmarshalVTUnsafe(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 15:
			if wireType != 2 {
>>>>>>> 672686754e (X-Smart-Squash: Squashed 5 commits:)
				return fmt.Errorf("proto: wrong wireType = %d for field Architecture", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return protohelpers.ErrIntOverflow
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return protohelpers.ErrInvalidLength
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return protohelpers.ErrInvalidLength
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			var stringValue string
			if intStringLen > 0 {
				stringValue = unsafe.String(&dAtA[iNdEx], intStringLen)
			}
			m.Architecture = stringValue
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := protohelpers.Skip(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return protohelpers.ErrInvalidLength
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.unknownFields = append(m.unknownFields, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
