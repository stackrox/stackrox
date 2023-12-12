package protoconvert

import (
	"github.com/stackrox/rox/generated/test"
	"github.com/stackrox/rox/generated/test2"
)

// ConvertSliceTestTestCloneInt32ToTest2TestCloneInt32 converts a slice of *test.TestClone_Int32 to a slice of *test2.TestClone_Int32
func ConvertSliceTestTestCloneInt32ToTest2TestCloneInt32(p1 []*test.TestClone_Int32) []*test2.TestClone_Int32 {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test2.TestClone_Int32, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTestTestCloneInt32ToTest2TestCloneInt32(v))
	}
	return p2
}

// ConvertTestTestCloneInt32ToTest2TestCloneInt32 converts from *test.TestClone_Int32 to *test2.TestClone_Int32
func ConvertTestTestCloneInt32ToTest2TestCloneInt32(p1 *test.TestClone_Int32) *test2.TestClone_Int32 {
	if p1 == nil {
		return nil
	}
	p2 := new(test2.TestClone_Int32)
	p2.Int32 = p1.Int32
	return p2
}

// ConvertSliceTestTestCloneMsgToTest2TestCloneMsg converts a slice of *test.TestClone_Msg to a slice of *test2.TestClone_Msg
func ConvertSliceTestTestCloneMsgToTest2TestCloneMsg(p1 []*test.TestClone_Msg) []*test2.TestClone_Msg {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test2.TestClone_Msg, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTestTestCloneMsgToTest2TestCloneMsg(v))
	}
	return p2
}

// ConvertTestTestCloneMsgToTest2TestCloneMsg converts from *test.TestClone_Msg to *test2.TestClone_Msg
func ConvertTestTestCloneMsgToTest2TestCloneMsg(p1 *test.TestClone_Msg) *test2.TestClone_Msg {
	if p1 == nil {
		return nil
	}
	p2 := new(test2.TestClone_Msg)
	p2.Msg = ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage(p1.Msg)
	return p2
}

// ConvertSliceTestTestCloneStringToTest2TestCloneString converts a slice of *test.TestClone_String_ to a slice of *test2.TestClone_String_
func ConvertSliceTestTestCloneStringToTest2TestCloneString(p1 []*test.TestClone_String_) []*test2.TestClone_String_ {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test2.TestClone_String_, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTestTestCloneStringToTest2TestCloneString(v))
	}
	return p2
}

// ConvertTestTestCloneStringToTest2TestCloneString converts from *test.TestClone_String_ to *test2.TestClone_String_
func ConvertTestTestCloneStringToTest2TestCloneString(p1 *test.TestClone_String_) *test2.TestClone_String_ {
	if p1 == nil {
		return nil
	}
	p2 := new(test2.TestClone_String_)
	p2.String_ = p1.String_
	return p2
}

// ConvertSliceTestTestCloneSubMessageToTest2TestCloneSubMessage converts a slice of *test.TestCloneSubMessage to a slice of *test2.TestCloneSubMessage
func ConvertSliceTestTestCloneSubMessageToTest2TestCloneSubMessage(p1 []*test.TestCloneSubMessage) []*test2.TestCloneSubMessage {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test2.TestCloneSubMessage, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage(v))
	}
	return p2
}

// ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage converts from *test.TestCloneSubMessage to *test2.TestCloneSubMessage
func ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage(p1 *test.TestCloneSubMessage) *test2.TestCloneSubMessage {
	if p1 == nil {
		return nil
	}
	p2 := new(test2.TestCloneSubMessage)
	p2.Int32 = p1.Int32
	p2.String_ = p1.String_
	return p2
}

// ConvertSliceTestTestCloneToTest2TestClone converts a slice of *test.TestClone to a slice of *test2.TestClone
func ConvertSliceTestTestCloneToTest2TestClone(p1 []*test.TestClone) []*test2.TestClone {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test2.TestClone, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTestTestCloneToTest2TestClone(v))
	}
	return p2
}

// ConvertTestTestCloneToTest2TestClone converts from *test.TestClone to *test2.TestClone
func ConvertTestTestCloneToTest2TestClone(p1 *test.TestClone) *test2.TestClone {
	if p1 == nil {
		return nil
	}
	p2 := new(test2.TestClone)
	if p1.IntSlice != nil {
		p2.IntSlice = make([]int32, len(p1.IntSlice))
		copy(p2.IntSlice, p1.IntSlice)
	}
	if p1.StringSlice != nil {
		p2.StringSlice = make([]string, len(p1.StringSlice))
		copy(p2.StringSlice, p1.StringSlice)
	}
	if p1.SubMessages != nil {
		p2.SubMessages = make([]*test2.TestCloneSubMessage, len(p1.SubMessages))
		for idx := range p1.SubMessages {
			p2.SubMessages[idx] = ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage(p1.SubMessages[idx])
		}
	}
	if p1.MessageMap != nil {
		p2.MessageMap = make(map[string]*test2.TestCloneSubMessage, len(p1.MessageMap))
		for k, v := range p1.MessageMap {
			p2.MessageMap[k] = ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage(v)
		}
	}
	if p1.StringMap != nil {
		p2.StringMap = make(map[string]string, len(p1.StringMap))
		for k, v := range p1.StringMap {
			p2.StringMap[k] = v
		}
	}
	if p1.EnumSlice != nil {
		p2.EnumSlice = make([]test2.TestClone_CloneEnum, len(p1.EnumSlice))
		for idx := range p1.EnumSlice {
			p2.EnumSlice[idx] = test2.TestClone_CloneEnum(p1.EnumSlice[idx])
		}
	}
	p2.Ts = p1.Ts.Clone()
	if p1.Primitive != nil {
		if val, ok := p1.Primitive.(*test.TestClone_Int32); ok {
			p2.Primitive = ConvertTestTestCloneInt32ToTest2TestCloneInt32(val)
		}
		if val, ok := p1.Primitive.(*test.TestClone_String_); ok {
			p2.Primitive = ConvertTestTestCloneStringToTest2TestCloneString(val)
		}
		if val, ok := p1.Primitive.(*test.TestClone_Msg); ok {
			p2.Primitive = ConvertTestTestCloneMsgToTest2TestCloneMsg(val)
		}
	}
	p2.Any = p1.Any.Clone()
	if p1.BytesMap != nil {
		p2.BytesMap = make(map[string][]uint8, len(p1.BytesMap))
		for k, v := range p1.BytesMap {
			p2.BytesMap[k] = make([]byte, len(v))
			copy(p2.BytesMap[k], v)
		}
	}
	if p1.BytesSlice != nil {
		p2.BytesSlice = make([][]uint8, len(p1.BytesSlice))
		for idx := range p1.BytesSlice {
			p2.BytesSlice[idx] = make([]byte, len(p1.BytesSlice[idx]))
			copy(p2.BytesSlice[idx], p1.BytesSlice[idx])
		}
	}
	if p1.Bytes != nil {
		p2.Bytes = make([]uint8, len(p1.Bytes))
		copy(p2.Bytes, p1.Bytes)
	}
	p2.SubMessage = ConvertTestTestCloneSubMessageToTest2TestCloneSubMessage(p1.SubMessage)
	p2.SingleEnum = test2.TestClone_CloneEnum(p1.SingleEnum)
	return p2
}

// ConvertSliceTest2TestCloneInt32ToTestTestCloneInt32 converts a slice of *test2.TestClone_Int32 to a slice of *test.TestClone_Int32
func ConvertSliceTest2TestCloneInt32ToTestTestCloneInt32(p1 []*test2.TestClone_Int32) []*test.TestClone_Int32 {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test.TestClone_Int32, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTest2TestCloneInt32ToTestTestCloneInt32(v))
	}
	return p2
}

// ConvertTest2TestCloneInt32ToTestTestCloneInt32 converts from *test2.TestClone_Int32 to *test.TestClone_Int32
func ConvertTest2TestCloneInt32ToTestTestCloneInt32(p1 *test2.TestClone_Int32) *test.TestClone_Int32 {
	if p1 == nil {
		return nil
	}
	p2 := new(test.TestClone_Int32)
	p2.Int32 = p1.Int32
	return p2
}

// ConvertSliceTest2TestCloneMsgToTestTestCloneMsg converts a slice of *test2.TestClone_Msg to a slice of *test.TestClone_Msg
func ConvertSliceTest2TestCloneMsgToTestTestCloneMsg(p1 []*test2.TestClone_Msg) []*test.TestClone_Msg {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test.TestClone_Msg, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTest2TestCloneMsgToTestTestCloneMsg(v))
	}
	return p2
}

// ConvertTest2TestCloneMsgToTestTestCloneMsg converts from *test2.TestClone_Msg to *test.TestClone_Msg
func ConvertTest2TestCloneMsgToTestTestCloneMsg(p1 *test2.TestClone_Msg) *test.TestClone_Msg {
	if p1 == nil {
		return nil
	}
	p2 := new(test.TestClone_Msg)
	p2.Msg = ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage(p1.Msg)
	return p2
}

// ConvertSliceTest2TestCloneStringToTestTestCloneString converts a slice of *test2.TestClone_String_ to a slice of *test.TestClone_String_
func ConvertSliceTest2TestCloneStringToTestTestCloneString(p1 []*test2.TestClone_String_) []*test.TestClone_String_ {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test.TestClone_String_, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTest2TestCloneStringToTestTestCloneString(v))
	}
	return p2
}

// ConvertTest2TestCloneStringToTestTestCloneString converts from *test2.TestClone_String_ to *test.TestClone_String_
func ConvertTest2TestCloneStringToTestTestCloneString(p1 *test2.TestClone_String_) *test.TestClone_String_ {
	if p1 == nil {
		return nil
	}
	p2 := new(test.TestClone_String_)
	p2.String_ = p1.String_
	return p2
}

// ConvertSliceTest2TestCloneSubMessageToTestTestCloneSubMessage converts a slice of *test2.TestCloneSubMessage to a slice of *test.TestCloneSubMessage
func ConvertSliceTest2TestCloneSubMessageToTestTestCloneSubMessage(p1 []*test2.TestCloneSubMessage) []*test.TestCloneSubMessage {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test.TestCloneSubMessage, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage(v))
	}
	return p2
}

// ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage converts from *test2.TestCloneSubMessage to *test.TestCloneSubMessage
func ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage(p1 *test2.TestCloneSubMessage) *test.TestCloneSubMessage {
	if p1 == nil {
		return nil
	}
	p2 := new(test.TestCloneSubMessage)
	p2.Int32 = p1.Int32
	p2.String_ = p1.String_
	return p2
}

// ConvertSliceTest2TestCloneToTestTestClone converts a slice of *test2.TestClone to a slice of *test.TestClone
func ConvertSliceTest2TestCloneToTestTestClone(p1 []*test2.TestClone) []*test.TestClone {
	if p1 == nil {
		return nil
	}
	p2 := make([]*test.TestClone, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertTest2TestCloneToTestTestClone(v))
	}
	return p2
}

// ConvertTest2TestCloneToTestTestClone converts from *test2.TestClone to *test.TestClone
func ConvertTest2TestCloneToTestTestClone(p1 *test2.TestClone) *test.TestClone {
	if p1 == nil {
		return nil
	}
	p2 := new(test.TestClone)
	if p1.IntSlice != nil {
		p2.IntSlice = make([]int32, len(p1.IntSlice))
		copy(p2.IntSlice, p1.IntSlice)
	}
	if p1.StringSlice != nil {
		p2.StringSlice = make([]string, len(p1.StringSlice))
		copy(p2.StringSlice, p1.StringSlice)
	}
	if p1.SubMessages != nil {
		p2.SubMessages = make([]*test.TestCloneSubMessage, len(p1.SubMessages))
		for idx := range p1.SubMessages {
			p2.SubMessages[idx] = ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage(p1.SubMessages[idx])
		}
	}
	if p1.MessageMap != nil {
		p2.MessageMap = make(map[string]*test.TestCloneSubMessage, len(p1.MessageMap))
		for k, v := range p1.MessageMap {
			p2.MessageMap[k] = ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage(v)
		}
	}
	if p1.StringMap != nil {
		p2.StringMap = make(map[string]string, len(p1.StringMap))
		for k, v := range p1.StringMap {
			p2.StringMap[k] = v
		}
	}
	if p1.EnumSlice != nil {
		p2.EnumSlice = make([]test.TestClone_CloneEnum, len(p1.EnumSlice))
		for idx := range p1.EnumSlice {
			p2.EnumSlice[idx] = test.TestClone_CloneEnum(p1.EnumSlice[idx])
		}
	}
	p2.Ts = p1.Ts.Clone()
	if p1.Primitive != nil {
		if val, ok := p1.Primitive.(*test2.TestClone_Int32); ok {
			p2.Primitive = ConvertTest2TestCloneInt32ToTestTestCloneInt32(val)
		}
		if val, ok := p1.Primitive.(*test2.TestClone_String_); ok {
			p2.Primitive = ConvertTest2TestCloneStringToTestTestCloneString(val)
		}
		if val, ok := p1.Primitive.(*test2.TestClone_Msg); ok {
			p2.Primitive = ConvertTest2TestCloneMsgToTestTestCloneMsg(val)
		}
	}
	p2.Any = p1.Any.Clone()
	if p1.BytesMap != nil {
		p2.BytesMap = make(map[string][]uint8, len(p1.BytesMap))
		for k, v := range p1.BytesMap {
			p2.BytesMap[k] = make([]byte, len(v))
			copy(p2.BytesMap[k], v)
		}
	}
	if p1.BytesSlice != nil {
		p2.BytesSlice = make([][]uint8, len(p1.BytesSlice))
		for idx := range p1.BytesSlice {
			p2.BytesSlice[idx] = make([]byte, len(p1.BytesSlice[idx]))
			copy(p2.BytesSlice[idx], p1.BytesSlice[idx])
		}
	}
	if p1.Bytes != nil {
		p2.Bytes = make([]uint8, len(p1.Bytes))
		copy(p2.Bytes, p1.Bytes)
	}
	p2.SubMessage = ConvertTest2TestCloneSubMessageToTestTestCloneSubMessage(p1.SubMessage)
	p2.SingleEnum = test.TestClone_CloneEnum(p1.SingleEnum)
	return p2
}

