
package schema

import (
	"fmt"
	"reflect"
<<<<<<< HEAD

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
=======
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
)

var (
	// CreateTableImageComponentV2Stmt holds the create statement for table `image_component_v2`.
	CreateTableImageComponentV2Stmt = &postgres.CreateStmts{
		GormModel: (*ImageComponentV2)(nil),
<<<<<<< HEAD
		Children:  []*postgres.CreateStmts{},
=======
		Children: []*postgres.CreateStmts{
			&postgres.CreateStmts{
				GormModel: (*ImageComponentV2Cves)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
	}

	// ImageComponentV2Schema is the go schema for table `image_component_v2`.
	ImageComponentV2Schema = func() *walker.Schema {
		schema := GetSchemaForTable("image_component_v2")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.ImageComponentV2)(nil)), "image_component_v2")
		referencedSchemas := map[string]*walker.Schema{
			"storage.Image": ImagesSchema,
		}

		schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
			return referencedSchemas[fmt.Sprintf("storage.%s", messageTypeName)]
		})
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_IMAGE_COMPONENTS_V2, "imagecomponentv2", (*storage.ImageComponentV2)(nil)))
		schema.SetSearchScope([]v1.SearchCategory{
<<<<<<< HEAD
			v1.SearchCategory_IMAGE_VULNERABILITIES_V2,
=======
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
			v1.SearchCategory_IMAGE_COMPONENTS_V2,
			v1.SearchCategory_IMAGES,
			v1.SearchCategory_DEPLOYMENTS,
			v1.SearchCategory_NAMESPACES,
			v1.SearchCategory_CLUSTERS,
		}...)
		schema.ScopingResource = resources.Image
<<<<<<< HEAD
		RegisterTable(schema, CreateTableImageComponentV2Stmt, features.FlattenCVEData.Enabled)
=======
		RegisterTable(schema, CreateTableImageComponentV2Stmt)
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
		mapping.RegisterCategoryToTable(v1.SearchCategory_IMAGE_COMPONENTS_V2, schema)
		return schema
	}()
)

const (
	// ImageComponentV2TableName specifies the name of the table in postgres.
	ImageComponentV2TableName = "image_component_v2"
<<<<<<< HEAD
=======
	// ImageComponentV2CvesTableName specifies the name of the table in postgres.
	ImageComponentV2CvesTableName = "image_component_v2_cves"
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
)

// ImageComponentV2 holds the Gorm model for Postgres table `image_component_v2`.
type ImageComponentV2 struct {
	ID              string             `gorm:"column:id;type:varchar;primaryKey"`
	Name            string             `gorm:"column:name;type:varchar"`
	Version         string             `gorm:"column:version;type:varchar"`
	Priority        int64              `gorm:"column:priority;type:bigint"`
	Source          storage.SourceType `gorm:"column:source;type:integer"`
	RiskScore       float32            `gorm:"column:riskscore;type:numeric"`
	TopCvss         float32            `gorm:"column:topcvss;type:numeric"`
	OperatingSystem string             `gorm:"column:operatingsystem;type:varchar"`
	ImageID         string             `gorm:"column:imageid;type:varchar;index:imagecomponentv2_imageid,type:btree"`
	Location        string             `gorm:"column:location;type:varchar"`
	Serialized      []byte             `gorm:"column:serialized;type:bytea"`
	ImagesRef       Images             `gorm:"foreignKey:imageid;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
<<<<<<< HEAD
=======

// ImageComponentV2Cves holds the Gorm model for Postgres table `image_component_v2_cves`.
type ImageComponentV2Cves struct {
<<<<<<< HEAD
	ImageComponentV2ID             string                        `gorm:"column:image_component_v2_id;type:varchar;primaryKey"`
	Idx                            int                           `gorm:"column:idx;type:integer;primaryKey;index:imagecomponentv2cves_idx,type:btree"`
	ID                             string                        `gorm:"column:id;type:varchar"`
	ImageID                        string                        `gorm:"column:imageid;type:varchar;index:imagecomponentv2cves_imageid,type:btree"`
	CveBaseInfoCve                 string                        `gorm:"column:cvebaseinfo_cve;type:varchar;index:imagecomponentv2cves_cvebaseinfo_cve,type:btree"`
	CveBaseInfoPublishedOn         *time.Time                    `gorm:"column:cvebaseinfo_publishedon;type:timestamp"`
	CveBaseInfoCreatedAt           *time.Time                    `gorm:"column:cvebaseinfo_createdat;type:timestamp"`
	CveBaseInfoEpssEpssProbability float32                       `gorm:"column:cvebaseinfo_epss_epssprobability;type:numeric"`
	OperatingSystem                string                        `gorm:"column:operatingsystem;type:varchar"`
	Cvss                           float32                       `gorm:"column:cvss;type:numeric"`
	Severity                       storage.VulnerabilitySeverity `gorm:"column:severity;type:integer;index:imagecomponentv2cves_severity,type:btree"`
	ImpactScore                    float32                       `gorm:"column:impactscore;type:numeric"`
	Nvdcvss                        float32                       `gorm:"column:nvdcvss;type:numeric"`
	FirstImageOccurrence           *time.Time                    `gorm:"column:firstimageoccurrence;type:timestamp"`
	State                          storage.VulnerabilityState    `gorm:"column:state;type:integer;index:imagecomponentv2cves_state,type:btree"`
	IsFixable                      bool                          `gorm:"column:isfixable;type:bool"`
	FixedBy                        string                        `gorm:"column:fixedby;type:varchar"`
	ImageComponentV2Ref            ImageComponentV2              `gorm:"foreignKey:image_component_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
	ImagesRef                      Images                        `gorm:"foreignKey:imageid;references:id;belongsTo;constraint:OnDelete:CASCADE"`
=======
	ImageComponentV2ID     string                        `gorm:"column:image_component_v2_id;type:varchar;primaryKey"`
	Idx                    int                           `gorm:"column:idx;type:integer;primaryKey;index:imagecomponentv2cves_idx,type:btree"`
	ID                     string                        `gorm:"column:id;type:varchar"`
	ImageID                string                        `gorm:"column:imageid;type:varchar;index:imagecomponentv2cves_imageid,type:btree"`
	CveBaseInfoCve         string                        `gorm:"column:cvebaseinfo_cve;type:varchar;index:imagecomponentv2cves_cvebaseinfo_cve,type:btree"`
	CveBaseInfoPublishedOn *time.Time                    `gorm:"column:cvebaseinfo_publishedon;type:timestamp"`
	CveBaseInfoCreatedAt   *time.Time                    `gorm:"column:cvebaseinfo_createdat;type:timestamp"`
	OperatingSystem        string                        `gorm:"column:operatingsystem;type:varchar"`
	Cvss                   float32                       `gorm:"column:cvss;type:numeric"`
	Severity               storage.VulnerabilitySeverity `gorm:"column:severity;type:integer;index:imagecomponentv2cves_severity,type:btree"`
	ImpactScore            float32                       `gorm:"column:impactscore;type:numeric"`
	Nvdcvss                float32                       `gorm:"column:nvdcvss;type:numeric"`
	FirstImageOccurrence   *time.Time                    `gorm:"column:firstimageoccurrence;type:timestamp"`
	State                  storage.VulnerabilityState    `gorm:"column:state;type:integer;index:imagecomponentv2cves_state,type:btree"`
	IsFixable              bool                          `gorm:"column:isfixable;type:bool"`
	FixedBy                string                        `gorm:"column:fixedby;type:varchar"`
	ImageComponentV2Ref    ImageComponentV2              `gorm:"foreignKey:image_component_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
	ImagesRef              Images                        `gorm:"foreignKey:imageid;references:id;belongsTo;constraint:OnDelete:CASCADE"`
>>>>>>> baec3e8b51 (X-Smart-Squash: Squashed 9 commits:)
}
>>>>>>> 9a652b9c15 (X-Smart-Squash: Squashed 10 commits:)
