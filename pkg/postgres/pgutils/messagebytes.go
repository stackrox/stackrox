package pgutils

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// MarshalRepeatedMessages marshals a slice of proto messages into a single []byte.
// Each message is length-delimited: varint(len) + marshaled bytes.
// This encoding matches proto's own wire format for length-delimited fields.
func MarshalRepeatedMessages[T interface{ MarshalVT() ([]byte, error) }](msgs []T) ([]byte, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	var buf []byte
	for _, msg := range msgs {
		data, err := msg.MarshalVT()
		if err != nil {
			return nil, err
		}
		buf = protowire.AppendVarint(buf, uint64(len(data)))
		buf = append(buf, data...)
	}
	return buf, nil
}

// MustMarshalRepeatedMessages is like MarshalRepeatedMessages but panics on error.
// Safe to use for well-formed proto messages in generated store code.
func MustMarshalRepeatedMessages[T interface{ MarshalVT() ([]byte, error) }](msgs []T) []byte {
	data, err := MarshalRepeatedMessages(msgs)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal repeated messages: %v", err))
	}
	return data
}

// UnmarshalRepeatedMessages unmarshals a []byte (produced by MarshalRepeatedMessages)
// back into a slice of proto messages. The newFn creates a new empty message instance.
func UnmarshalRepeatedMessages[T interface{ UnmarshalVT([]byte) error }](data []byte, newFn func() T) ([]T, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var msgs []T
	for len(data) > 0 {
		size, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return nil, fmt.Errorf("messagebytes: invalid varint at offset %d", len(data))
		}
		data = data[n:]
		if uint64(len(data)) < size {
			return nil, fmt.Errorf("messagebytes: truncated message, need %d bytes but only %d remain", size, len(data))
		}
		msg := newFn()
		if err := msg.UnmarshalVT(data[:size]); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
		data = data[size:]
	}
	return msgs, nil
}

// MustUnmarshalRepeatedMessages is like UnmarshalRepeatedMessages but panics on error.
// Safe to use in generated store scanner code where data integrity is guaranteed by the DB.
func MustUnmarshalRepeatedMessages[T interface{ UnmarshalVT([]byte) error }](data []byte, newFn func() T) []T {
	msgs, err := UnmarshalRepeatedMessages(data, newFn)
	if err != nil {
		panic(fmt.Sprintf("failed to unmarshal repeated messages: %v", err))
	}
	return msgs
}
