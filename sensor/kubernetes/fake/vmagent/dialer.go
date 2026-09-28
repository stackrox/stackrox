package vmagent

import (
	"context"
	"fmt"
	"io"

	"github.com/stackrox/rox/sensor/common/virtualmachine/vmscraper"
)

var _ vmscraper.VMDialer = (*Dialer)(nil)

// Dialer is a VMDialer that returns a nop stream instead of opening VSOCK.
type Dialer struct{}

// NewDialer returns a Dialer.
func NewDialer() *Dialer {
	return &Dialer{}
}

// Dial returns a nop stream so VMScraper can run its poll path without VSOCK.
func (*Dialer) Dial(ctx context.Context, _, _ string, _ uint32, _ bool) (io.ReadWriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("fake vm dial: %w", err)
	}
	return nopStream{}, nil
}

type nopStream struct{}

func (nopStream) Read([]byte) (int, error) { return 0, io.EOF }

func (nopStream) Write(p []byte) (int, error) { return len(p), nil }

func (nopStream) Close() error { return nil }
