package puller

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

type fakeStore struct {
	vms []*virtualmachine.Info
}

func (s *fakeStore) ListRunning() []*virtualmachine.Info { return s.vms }

type fakeDialer struct {
	reports map[string]*v1.VMReport // key: "namespace/name"
	err     error
}

func (d *fakeDialer) Dial(namespace, name string, _ uint32, _ bool) (vsockclient.StreamReader, error) {
	if d.err != nil {
		return nil, d.err
	}
	key := namespace + "/" + name
	report, ok := d.reports[key]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	data, _ := proto.Marshal(report)
	return &fakeStream{Reader: bytes.NewReader(data)}, nil
}

type fakeStream struct {
	io.Reader
}

func (f *fakeStream) Close() error { return nil }

type fakeSender struct {
	received []*v1.IndexReport
}

func (s *fakeSender) Send(_ context.Context, report *v1.IndexReport) error {
	s.received = append(s.received, report)
	return nil
}

func TestPuller_PollSendsReports(t *testing.T) {
	report := &v1.VMReport{
		IndexReport: &v1.IndexReport{
			VsockCid: "42",
			IndexV4:  &v4.IndexReport{Success: true},
		},
	}

	store := &fakeStore{vms: []*virtualmachine.Info{
		{ID: "vm-1", Name: "test-vm", Namespace: "default", Running: true},
	}}
	dialer := &fakeDialer{reports: map[string]*v1.VMReport{
		"default/test-vm": report,
	}}
	sender := &fakeSender{}

	p := New(store, sender, dialer)

	p.poll()

	require.Len(t, sender.received, 1)
	assert.Equal(t, "42", sender.received[0].GetVsockCid())
	assert.True(t, sender.received[0].GetIndexV4().GetSuccess())
}

func TestPuller_PollSkipsUnreachableVMs(t *testing.T) {
	store := &fakeStore{vms: []*virtualmachine.Info{
		{ID: "vm-ok", Name: "reachable", Namespace: "ns1", Running: true},
		{ID: "vm-bad", Name: "unreachable", Namespace: "ns2", Running: true},
	}}
	dialer := &fakeDialer{reports: map[string]*v1.VMReport{
		"ns1/reachable": {
			IndexReport: &v1.IndexReport{
				VsockCid: "10",
				IndexV4:  &v4.IndexReport{Success: true},
			},
		},
		// "ns2/unreachable" is not in the map → dial returns error
	}}
	sender := &fakeSender{}

	p := New(store, sender, dialer)
	p.poll()

	require.Len(t, sender.received, 1, "should only send the reachable VM's report")
	assert.Equal(t, "10", sender.received[0].GetVsockCid())
}

func TestPuller_PollNoRunningVMs(t *testing.T) {
	store := &fakeStore{vms: nil}
	sender := &fakeSender{}
	p := New(store, sender, &fakeDialer{})

	p.poll()

	assert.Empty(t, sender.received)
}

func TestPuller_StartStop(t *testing.T) {
	store := &fakeStore{}
	sender := &fakeSender{}
	p := New(store, sender, &fakeDialer{})
	p.interval = 50 * time.Millisecond

	require.NoError(t, p.Start())
	time.Sleep(100 * time.Millisecond)
	p.Stop()
}
