// Code generated by protoc-gen-go-compat. DO NOT EDIT.

package test

func (m *TestCloneSubMessage) Size() int                   { return m.SizeVT() }
func (m *TestCloneSubMessage) Clone() *TestCloneSubMessage { return m.CloneVT() }
func (m *TestCloneSubMessage) Marshal() ([]byte, error)    { return m.MarshalVT() }
func (m *TestCloneSubMessage) Unmarshal(dAtA []byte) error { return m.UnmarshalVT(dAtA) }

func (m *TestClone) Size() int                   { return m.SizeVT() }
func (m *TestClone) Clone() *TestClone           { return m.CloneVT() }
func (m *TestClone) Marshal() ([]byte, error)    { return m.MarshalVT() }
func (m *TestClone) Unmarshal(dAtA []byte) error { return m.UnmarshalVT(dAtA) }