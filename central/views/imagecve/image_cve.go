package imagecve

import (
	"time"
)

type imageCVECore struct {
	CVE                     string    `db:"cve"`
	TopCVSS                 float32   `db:"cvss_max"`
	AffectedImages          int       `db:"image_sha_count"`
	FirstDiscoveredInSystem time.Time `db:"cve_created_time_min"`
}

func (c *imageCVECore) GetCVE() string {
	return c.CVE
}

func (c *imageCVECore) GetTopCVSS() float32 {
	return c.TopCVSS
}

func (c *imageCVECore) GetAffectedImages() int {
	return c.AffectedImages
}

func (c *imageCVECore) GetFirstDiscoveredInSystem() time.Time {
	return c.FirstDiscoveredInSystem
}
