package models

import (
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
)

const (
	AlertsTableName = "alerts"
)

/*
type Policy struct {
	Id                 string `gorm:"type:varchar"`
	Name               string `gorm:"type:varchar"`
	Description        string `gorm:"type:varchar"`
	Disabled           bool
	Categories         []string         `gorm:"type:text[]"`
	LifecycleStages    []int            `gorm:"type:integer[];column:lifecyclestages"`
	Severity           int              `gorm:"type:integer"`
	EnforcementActions []int            `gorm:"type:integer[];column:enforcementactions"`
	LastUpdated        *types.Timestamp `gorm:"type:timestamp;column:lastupdated"`
	SORTName           string           `gorm:"type:varchar;column:sortname"`
	SORTLifecycleStage string           `gorm:"type:varchar;column:sortlifecyclestage"`
	SORTEnforcement    bool             `gorm:"column:sortenforcement"`
}

type Deployment struct {
	Id          string `gorm:"type:varchar;index:alerts_deployment_id,type:hash"`
	Name        string `gorm:"type:varchar"`
	Namespace   string `gorm:"type:varchar"`
	NamespaceId string `gorm:"type:varchar;column:namespaceid"`
	ClusterId   string `gorm:"type:varchar;column:clusterid"`
	ClusterName string `gorm:"type:varchar;column:clustername"`
	Inactive    bool
}

type ImageName struct {
	Registry string `gorm:"type:varchar"`
	Remote   string `gorm:"type:varchar"`
	Tag      string `gorm:"type:varchar"`
	FullName string `gorm:"type:varchar;column:fullname"`
}

type Resource struct {
	ResourceType int    `gorm:"type:integer;column:resourcetype"`
	Name         string `gorm:"type:varchar"`
}

type ContainerImage struct {
	Id   string    `gorm:"type:varchar"`
	Name ImageName `gorm:"embedded;embeddedPrefix:name_"`
}

type Alerts struct {
	Id                string           `gorm:"type:varchar;primarykey"`
	Serialized        string           `gorm:"type:bytea"`
	Policy            Policy           `gorm:"embedded;embeddedPrefix:policy_"`
	LifecycleStage    int              `gorm:"type:integer;column:lifecyclestage;index:alerts_lifecyclestage,type:btree"`
	ClusterId         string           `gorm:"type:varchar;column:clusterid"`
	ClusterName       string           `gorm:"type:varchar;column:clustername"`
	Namespace         string           `gorm:"type:varchar;column:namespace"`
	NamespaceId       string           `gorm:"type:varchar;column:namespaceid"`
	Deployment        Deployment       `gorm:"embedded;embeddedPrefix:deployment_"`
	Image             *ContainerImage  `gorm:"embedded;embeddedPrefix:image_"`
	EnforcementAction int              `gorm:"type:integer"`
	Time              *types.Timestamp `gorm:"type:timestamp"`
	State             int              `gorm:"type:integer;index:alerts_state,type:btree"`
	Tags              []string         `gorm:"type:text[]"`
	Resource          Resource         `gorm:"embedded;embeddedPrefix:resource_"`
}
*/

type Alert struct {
	Id         string `gorm:"type:varchar;primarykey"`
	Serialized []byte `gorm:"type:bytea"`

	PolicyId                 string           `gorm:"type:varchar;column:policy_id"`
	PolicyName               string           `gorm:"type:varchar;column:policy_name"`
	PolicyDescription        string           `gorm:"type:varchar;column:policy_description"`
	PolicyDisabled           bool             `gorm:"column:policy_disabled"`
	PolicyCategories         *pq.StringArray  `gorm:"type:text[];column:policy_categories"`
	PolicyLifecycleStages    *pq.Int32Array   `gorm:"type:integer[];column:policy_lifecyclestages"`
	PolicySeverity           storage.Severity `gorm:"type:integer;column:policy_severity"`
	PolicyEnforcementActions *pq.Int32Array   `gorm:"type:integer[];column:policy_enforcementactions"`
	PolicyLastUpdated        *time.Time       `gorm:"type:timestamp;column:policy_lastupdated"`
	PolicySORTName           string           `gorm:"type:varchar;column:policy_sortname"`
	PolicySORTLifecycleStage string           `gorm:"type:varchar;column:policy_sortlifecyclestage"`
	PolicySORTEnforcement    bool             `gorm:"column:policy_sortenforcement"`

	LifecycleStage storage.LifecycleStage `gorm:"type:integer;column:lifecyclestage;index:alerts_lifecyclestage,type:btree"`
	ClusterId      string                 `gorm:"type:varchar;column:clusterid"`
	ClusterName    string                 `gorm:"type:varchar;column:clustername"`
	Namespace      string                 `gorm:"type:varchar;column:namespace"`
	NamespaceId    string                 `gorm:"type:varchar;column:namespaceid"`

	DeploymentId          string `gorm:"column:deployment_id;type:varchar;index:alerts_deployment_id,type:hash"`
	DeploymentName        string `gorm:"column:deployment_name;type:varchar"`
	DeploymentNamespace   string `gorm:"column:deployment_namespace;type:varchar"`
	DeploymentNamespaceId string `gorm:"type:varchar;column:deployment_namespaceid"`
	DeploymentClusterId   string `gorm:"type:varchar;column:deployment_clusterid"`
	DeploymentClusterName string `gorm:"type:varchar;column:deployment_clustername"`
	DeploymentInactive    bool   `gorm:"column:deployment_inactive"`

	ImageId           string `gorm:"type:varchar;column:image_id"`
	ImageNameRegistry string `gorm:"type:varchar;column:image_name_registry"`
	ImageNameRemote   string `gorm:"type:varchar;column:image_name_remote"`
	ImageNameTag      string `gorm:"type:varchar;column:image_name_tag"`
	ImageNameFullName string `gorm:"type:varchar;column:image_name_fullname"`

	ResourceResourceType storage.Alert_Resource_ResourceType `gorm:"type:integer;column:resource_resourcetype"`
	ResourceName         string                              `gorm:"type:varchar"`

	EnforcementAction storage.EnforcementAction `gorm:"type:integer"`
	Time              *time.Time                `gorm:"type:timestamp"`
	State             storage.ViolationState    `gorm:"type:integer;index:alerts_state,type:btree"`
	Tags              *pq.StringArray           `gorm:"type:text[]"`
}
