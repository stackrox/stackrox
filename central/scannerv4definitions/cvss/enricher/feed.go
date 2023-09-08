package enricher

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

// This is an envelope type so we can get at the cvssv3 object way in there.
type cve struct {
	CVE struct {
		Meta struct {
			ID string `json:"ID"`
		} `json:"CVE_data_meta"`
	} `json:"cve"`
	Impact struct {
		V3 struct {
			CVSS json.RawMessage `json:"cvssV3"`
		} `json:"baseMetricV3"`
	} `json:"impact"`
}

const bufferSize = 2048 // 2KB

func ProcessAndWriteCVSS(yr int, ctx context.Context, r io.Reader, file *os.File) error {
	ctx = zlog.ContextWithValues(ctx, "component", "central/scannerV4Definitions/cvss/ProcessAndWriteCVSS")
	bufReader := bufio.NewReaderSize(r, bufferSize)
	dec := json.NewDecoder(bufReader)
	bufWriter := bufio.NewWriterSize(file, bufferSize)
	enc := json.NewEncoder(bufWriter)
	var skip, wrote uint
	for {
		t, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for the start of the CVE_Items array
		if key, ok := t.(string); ok && key == "CVE_Items" {
			// Expect the next token to be the opening of the array
			t, err = dec.Token()
			if err != nil {
				return err
			}
			if delim, ok := t.(json.Delim); !ok || delim != '[' {
				return fmt.Errorf("expected start of CVE_Items array, got %v", t)
			}

			// Now, process each item in the CVE_Items array
			for dec.More() {
				var item cve
				if err := dec.Decode(&item); err != nil {
					return err
				}

				// Check if CVSS data is available, and if so, write to file
				if item.Impact.V3.CVSS != nil {
					r := driver.EnrichmentRecord{
						Tags:       []string{item.CVE.Meta.ID},
						Enrichment: item.Impact.V3.CVSS,
					}
					if err := enc.Encode(&r); err != nil {
						return fmt.Errorf("error encoding item with CVE ID %s: %w", item.CVE.Meta.ID, err)
					}
					// Flush after each item
					if err := bufWriter.Flush(); err != nil {
						return fmt.Errorf("error flushing writer after encoding item with CVE ID %s: %w", item.CVE.Meta.ID, err)
					}
					wrote++
				} else {
					skip++
					continue
				}
			}
			zlog.Debug(ctx).
				Uint("year", uint(yr)).
				Uint("skip", skip).
				Uint("wrote", wrote).
				Msg("wrote cvss items")
			// Once done processing the array, break out of the loop
			break
		}
	}
	return bufWriter.Flush()
}
