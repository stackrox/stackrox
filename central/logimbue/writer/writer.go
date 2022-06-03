package writer

import (
	"io"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/jsonutil"
)

// WriteLogs takes the LogImbue logs from the Store and writes them to Writer.
func WriteLogs(w io.Writer, logs []*storage.LogImbue) error {
	// Each log will be a JSON object. For convenience, we wrap it in "[]" so that
	// it is readable as a JSON array.
	jsonWriter := jsonutil.NewJSONArrayWriter(w)
	err := jsonWriter.Init()
	if err != nil {
		return err
	}
	for _, alog := range logs {
		err = jsonWriter.WriteObject(string(alog.Log))
		if err != nil {
			return err
		}
	}
	err = jsonWriter.Finish()
	return err
}
