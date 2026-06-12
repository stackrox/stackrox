package pgutils

import (
	"encoding/binary"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// MarshalRepeatedMessages serializes a slice of proto messages into a single bytea.
// Format: [4-byte big-endian length][message bytes] repeated. Returns nil for empty/nil slices.
func MarshalRepeatedMessages[T proto.Message](msgs []T) ([]byte, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	var buf []byte
	for _, msg := range msgs {
		data, err := proto.Marshal(msg)
		if err != nil {
			return nil, errors.Wrap(err, "marshaling repeated message")
		}
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
		buf = append(buf, lenBuf...)
		buf = append(buf, data...)
	}
	return buf, nil
}

// MustMarshalRepeatedMessages is like MarshalRepeatedMessages but panics on error.
func MustMarshalRepeatedMessages[T proto.Message](msgs []T) []byte {
	data, err := MarshalRepeatedMessages(msgs)
	if err != nil {
		panic(err)
	}
	return data
}

// UnmarshalRepeatedMessages deserializes a bytea produced by MarshalRepeatedMessages
// back into a slice of proto messages.
func UnmarshalRepeatedMessages[T proto.Message](data []byte, newMsg func() T) ([]T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var msgs []T
	for len(data) > 0 {
		if len(data) < 4 {
			return nil, errors.New("messagebytes: truncated length prefix")
		}
		length := binary.BigEndian.Uint32(data[:4])
		data = data[4:]
		if uint32(len(data)) < length {
			return nil, errors.New("messagebytes: truncated message data")
		}
		msg := newMsg()
		if err := proto.Unmarshal(data[:length], msg); err != nil {
			return nil, errors.Wrap(err, "unmarshaling repeated message")
		}
		msgs = append(msgs, msg)
		data = data[length:]
	}
	return msgs, nil
}
