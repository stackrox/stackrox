package enricher

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

type meta struct {
	LastModifiedDate time.Time
	Size             int64
	ZipSize          int64
	GZSize           int64
	SHA256           string
}

func (c *meta) parseBufferToMeta(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := strings.TrimSpace(parts[1])

		var err error
		switch key {
		case "lastModifiedDate":
			c.LastModifiedDate, err = time.Parse(time.RFC3339, value)
		case "size":
			_, err = fmt.Sscan(value, &c.Size)
		case "zipSize":
			_, err = fmt.Sscan(value, &c.ZipSize)
		case "gzSize":
			_, err = fmt.Sscan(value, &c.GZSize)
		case "sha256":
			c.SHA256 = value
		}
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}
