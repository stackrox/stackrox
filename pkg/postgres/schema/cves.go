package schema

import (
	"time"

	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableCvesStmt holds the create statement for table `cves`.
	CreateTableCvesStmt = &postgres.CreateStmts{
		GormModel: (*Cves)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// CvesTableName specifies the name of the table in postgres.
	CvesTableName = "cves"
)

// Cves holds the Gorm model for Postgres table `cves`.
type Cves struct {
	ID            string     `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	CveName       string     `gorm:"column:cve_name;type:text;not null"`
	Source        string     `gorm:"column:source;type:text;not null"`
	Severity      string     `gorm:"column:severity;type:text;not null"`
	CvssV2        *float32   `gorm:"column:cvss_v2;type:real"`
	CvssV3        *float32   `gorm:"column:cvss_v3;type:real"`
	NvdCvssV3     *float32   `gorm:"column:nvd_cvss_v3;type:real"`
	Summary       *string    `gorm:"column:summary;type:text"`
	Link          *string    `gorm:"column:link;type:text"`
	PublishedOn   *time.Time `gorm:"column:published_on;type:timestamptz"`
	AdvisoryName  *string    `gorm:"column:advisory_name;type:text"`
	AdvisoryLink  *string    `gorm:"column:advisory_link;type:text"`
	ContentHash   string     `gorm:"column:content_hash;type:text;not null"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;not null;default:now()"`
}
