package models

const (
	IntegrationHealthTableName = "integrationhealth"
)

type IntegrationHealth struct {
	Id         string `gorm:"type:varchar;primarykey"`
	Serialized []byte `gorm:"type:bytea"`
}
