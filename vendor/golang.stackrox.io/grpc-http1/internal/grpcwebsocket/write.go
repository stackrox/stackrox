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

package grpcwebsocket

import (
	"bytes"
	"context"
	"io"

	"github.com/golang/glog"
	"golang.stackrox.io/grpc-http1/internal/grpcproto"
	"golang.stackrox.io/grpc-http1/internal/ioutils"
	"nhooyr.io/websocket"
)

// Write the contents of the reader along the WebSocket connection.
// This is done by sending each WebSocket message as a gRPC message frame.
// Each message frame is length-prefixed message, where the prefix is 5 bytes.
// gRPC request format is specified here: https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md.
func Write(ctx context.Context, conn *websocket.Conn, r io.Reader, sender string) error {
	var msg bytes.Buffer
	for {
		// Reset the message buffer to start with a clean slate.
		msg.Reset()
		// Read message header into the msg buffer.
		if _, err := ioutils.CopyNFull(&msg, r, grpcproto.MessageHeaderLength); err != nil {
			if err == io.EOF {
				// EOF here means the sender has no more messages to send.
				return nil
			}

			glog.V(2).Infof("Malformed gRPC message when reading header sent from %s: %v", sender, err)
			return err
		}

		_, length, err := grpcproto.ParseMessageHeader(msg.Bytes())
		if err != nil {
			return err
		}

		// Read the rest of the message into the msg buffer.
		if n, err := io.CopyN(&msg, r, int64(length)); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				err = io.ErrUnexpectedEOF
				glog.V(2).Infof("Malformed gRPC message: fewer than the announced %d bytes in payload %s wants to send: %d", length, sender, n)
			} else {
				glog.V(2).Infof("Unable to read gRPC message %s wants to send: %v", sender, err)
			}
			return err
		}

		// Write the entire message frame along the WebSocket connection.
		if err := conn.Write(ctx, websocket.MessageBinary, msg.Bytes()); err != nil {
			glog.V(2).Infof("Unable to write gRPC message from %s: %v", sender, err)
			return err
		}
	}
}
