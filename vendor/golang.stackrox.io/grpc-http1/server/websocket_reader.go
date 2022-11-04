// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package server

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	"golang.stackrox.io/grpc-http1/internal/grpcproto"
	"nhooyr.io/websocket"
)

// readerResult stores the output from calls to (*wsReader).conn.Reader
// to be sent along a channel.
type readerResult struct {
	reader io.Reader
	err    error
}

// wsReader is an io.ReadCloser that wraps around a WebSocket's io.Reader.
type wsReader struct {
	ctx     context.Context
	conn    *websocket.Conn
	currMsg []byte

	// These are to prevent the WebSocket from closing due to
	// (*websocket.Conn).Reader's context potentially expiring.
	// This can happen if Read waits indefinitely, which we prevent
	// by decoupling the (*websocket.Conn).Reader and Read.
	// The context is required to let readerLoop and Read know to stop.
	readCtx       context.Context
	readCtxCancel context.CancelFunc
	readerResultC chan readerResult

	// We use (*websocket.Conn).Reader instead of (*websocket.Conn).Read
	// to remove the need for constant memory (de-)allocation when making
	// a new buffer per read. Instead, we choose to manage a single buffer.
	// The barrier is required to ensure there is only one reader used at-a-time.
	buf      bytes.Buffer
	barrierC chan struct{}

	// Errors should be "sticky".
	err error
}

func newWebSocketReader(ctx context.Context, conn *websocket.Conn) io.ReadCloser {
	r := &wsReader{
		ctx:           ctx,
		conn:          conn,
		readerResultC: make(chan readerResult),
		barrierC:      make(chan struct{}, 1),
	}
	r.barrierC <- struct{}{}
	r.readCtx, r.readCtxCancel = context.WithCancel(r.ctx)
	go r.readerLoop()
	return r
}

// readerLoop continuously obtains an io.Reader from the WebSocket connection and forwards each along the results channel.
// There may only be one io.Reader at-a-time.
func (r *wsReader) readerLoop() {
	for {
		select {
		case <-r.readCtx.Done():
			return
		case <-r.barrierC:
		}

		mt, reader, err := r.conn.Reader(r.ctx)
		if err == nil && mt != websocket.MessageBinary {
			err = errors.Errorf("incorrect message type; expected MessageBinary but got %v", mt)
			reader = nil
		}

		select {
		case <-r.readCtx.Done():
			return
		case r.readerResultC <- readerResult{
			reader: reader,
			err:    err,
		}:
		}
	}
}

// Read reads from the WebSocket connection.
// Read assumes each WebSocket message is a gRPC message or metadata frame.
func (r *wsReader) Read(p []byte) (int, error) {
	var n int
	// Errors are "sticky", so if we've errored before, don't bother reading.
	if r.err == nil {
		n, r.err = r.doRead(p)
	}
	return n, r.err
}

func (r *wsReader) doRead(p []byte) (int, error) {
	if len(r.currMsg) == 0 {
		var rr readerResult
		select {
		case <-r.readCtx.Done():
			// CloseRead was called or the request's context expired.
			// This is typically done in an error-case, only.
			return 0, errors.Wrap(r.readCtx.Err(), "reading websocket message")
		case rr = <-r.readerResultC:
		}

		if rr.err != nil {
			return 0, rr.err
		}

		r.buf.Reset()
		if _, err := r.buf.ReadFrom(rr.reader); err != nil {
			return 0, err
		}

		// Allow (*wsReader).readerLoop to get a new reader.
		r.barrierC <- struct{}{}

		// Expect either an EOS message from the client or a valid data frame.
		// Headers are not expected to be handled here.
		msg := r.buf.Bytes()
		if err := grpcproto.ValidateGRPCFrame(msg); err != nil {
			return 0, err
		}
		if grpcproto.IsEndOfStream(msg) {
			// This is where a connection without errors will terminate.
			return 0, io.EOF
		}
		if !grpcproto.IsDataFrame(msg) {
			return 0, errors.Errorf("message is not a gRPC data frame")
		}

		r.currMsg = msg
	}

	n := copy(p, r.currMsg)
	r.currMsg = r.currMsg[n:]

	return n, nil
}

// Close signals readerLoop that we are no longer accepting messages.
func (r *wsReader) Close() error {
	// We cannot call (*websocket.Conn).CloseRead here. The WebSocket's closing handshake
	// may not have been called yet, so the client may not know to stop sending messages.
	// If the client sends a message after we call (*websocket.Conn).CloseRead, the WebSocket
	// connection will return errors.
	// Instead, we cancel our read context to signal all reads to stop.
	r.readCtxCancel()
	return nil
}
