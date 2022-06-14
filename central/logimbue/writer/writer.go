package writer

import (
	"encoding/json"
	"io"

	"github.com/stackrox/stackrox/pkg/jsonutil"
)

type safeRawMessage []byte

// MarshalJSON returns m as the JSON encoding of m.
func (m safeRawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	msg, err := json.Marshal(json.RawMessage(m))
	if err != nil {
		msg, err = json.Marshal(map[string]interface{}{"encodingError": err.Error(), "raw": string(m)})
	}
	return msg, err
}

// WriteLogs takes the LogImbue logs from the Store and writes them to Writer.
func WriteLogs(w io.Writer, logs []string) error {
	// Each log will be a JSON object. For convenience, we wrap it in "[]" so that
	// it is readable as a JSON array.
	jsonWriter := jsonutil.NewJSONArrayWriter(w)
	err := jsonWriter.Init()
	if err != nil {
		return err
	}
	for _, alog := range logs {
		err = jsonWriter.WriteObject(safeRawMessage(alog))
		if err != nil {
			return err
		}
	}
	err = jsonWriter.Finish()
	return err
}
