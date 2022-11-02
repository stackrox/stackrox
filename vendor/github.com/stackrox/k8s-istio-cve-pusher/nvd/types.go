package nvd

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
)

func Load(data []byte) (*schema.NVDCVEFeedJSON10, error) {
	return LoadReader(bytes.NewReader(data))
}

func LoadReader(stream io.Reader) (*schema.NVDCVEFeedJSON10, error) {
	var cveFeed schema.NVDCVEFeedJSON10
	if err := json.NewDecoder(stream).Decode(&cveFeed); err != nil {
		return nil, err
	}
	return &cveFeed, nil
}
