package protoconvert

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// ConvertSliceStorageImageIntegrationClairToV1ImageIntegrationClair converts a slice of *storage.ImageIntegration_Clair to a slice of *v1.ImageIntegration_Clair
func ConvertSliceStorageImageIntegrationClairToV1ImageIntegrationClair(p1 []*storage.ImageIntegration_Clair) []*v1.ImageIntegration_Clair {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Clair, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationClairToV1ImageIntegrationClair(v))
	}
	return p2
}

// ConvertStorageImageIntegrationClairToV1ImageIntegrationClair converts from *storage.ImageIntegration_Clair to *v1.ImageIntegration_Clair
func ConvertStorageImageIntegrationClairToV1ImageIntegrationClair(p1 *storage.ImageIntegration_Clair) *v1.ImageIntegration_Clair {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Clair)
	p2.Clair = ConvertStorageClairConfigToV1ClairConfig(p1.Clair)
	return p2
}

// ConvertSliceStorageImageIntegrationClairV4ToV1ImageIntegrationClairV4 converts a slice of *storage.ImageIntegration_ClairV4 to a slice of *v1.ImageIntegration_ClairV4
func ConvertSliceStorageImageIntegrationClairV4ToV1ImageIntegrationClairV4(p1 []*storage.ImageIntegration_ClairV4) []*v1.ImageIntegration_ClairV4 {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_ClairV4, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationClairV4ToV1ImageIntegrationClairV4(v))
	}
	return p2
}

// ConvertStorageImageIntegrationClairV4ToV1ImageIntegrationClairV4 converts from *storage.ImageIntegration_ClairV4 to *v1.ImageIntegration_ClairV4
func ConvertStorageImageIntegrationClairV4ToV1ImageIntegrationClairV4(p1 *storage.ImageIntegration_ClairV4) *v1.ImageIntegration_ClairV4 {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_ClairV4)
	p2.ClairV4 = ConvertStorageClairV4ConfigToV1ClairV4Config(p1.ClairV4)
	return p2
}

// ConvertSliceStorageImageIntegrationClairifyToV1ImageIntegrationClairify converts a slice of *storage.ImageIntegration_Clairify to a slice of *v1.ImageIntegration_Clairify
func ConvertSliceStorageImageIntegrationClairifyToV1ImageIntegrationClairify(p1 []*storage.ImageIntegration_Clairify) []*v1.ImageIntegration_Clairify {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Clairify, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationClairifyToV1ImageIntegrationClairify(v))
	}
	return p2
}

// ConvertStorageImageIntegrationClairifyToV1ImageIntegrationClairify converts from *storage.ImageIntegration_Clairify to *v1.ImageIntegration_Clairify
func ConvertStorageImageIntegrationClairifyToV1ImageIntegrationClairify(p1 *storage.ImageIntegration_Clairify) *v1.ImageIntegration_Clairify {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Clairify)
	p2.Clairify = ConvertStorageClairifyConfigToV1ClairifyConfig(p1.Clairify)
	return p2
}

// ConvertSliceStorageImageIntegrationDockerToV1ImageIntegrationDocker converts a slice of *storage.ImageIntegration_Docker to a slice of *v1.ImageIntegration_Docker
func ConvertSliceStorageImageIntegrationDockerToV1ImageIntegrationDocker(p1 []*storage.ImageIntegration_Docker) []*v1.ImageIntegration_Docker {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Docker, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationDockerToV1ImageIntegrationDocker(v))
	}
	return p2
}

// ConvertStorageImageIntegrationDockerToV1ImageIntegrationDocker converts from *storage.ImageIntegration_Docker to *v1.ImageIntegration_Docker
func ConvertStorageImageIntegrationDockerToV1ImageIntegrationDocker(p1 *storage.ImageIntegration_Docker) *v1.ImageIntegration_Docker {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Docker)
	p2.Docker = ConvertStorageDockerConfigToV1DockerConfig(p1.Docker)
	return p2
}

// ConvertSliceStorageImageIntegrationEcrToV1ImageIntegrationEcr converts a slice of *storage.ImageIntegration_Ecr to a slice of *v1.ImageIntegration_Ecr
func ConvertSliceStorageImageIntegrationEcrToV1ImageIntegrationEcr(p1 []*storage.ImageIntegration_Ecr) []*v1.ImageIntegration_Ecr {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Ecr, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationEcrToV1ImageIntegrationEcr(v))
	}
	return p2
}

// ConvertStorageImageIntegrationEcrToV1ImageIntegrationEcr converts from *storage.ImageIntegration_Ecr to *v1.ImageIntegration_Ecr
func ConvertStorageImageIntegrationEcrToV1ImageIntegrationEcr(p1 *storage.ImageIntegration_Ecr) *v1.ImageIntegration_Ecr {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Ecr)
	p2.Ecr = ConvertStorageECRConfigToV1ECRConfig(p1.Ecr)
	return p2
}

// ConvertSliceStorageImageIntegrationGoogleToV1ImageIntegrationGoogle converts a slice of *storage.ImageIntegration_Google to a slice of *v1.ImageIntegration_Google
func ConvertSliceStorageImageIntegrationGoogleToV1ImageIntegrationGoogle(p1 []*storage.ImageIntegration_Google) []*v1.ImageIntegration_Google {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Google, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationGoogleToV1ImageIntegrationGoogle(v))
	}
	return p2
}

// ConvertStorageImageIntegrationGoogleToV1ImageIntegrationGoogle converts from *storage.ImageIntegration_Google to *v1.ImageIntegration_Google
func ConvertStorageImageIntegrationGoogleToV1ImageIntegrationGoogle(p1 *storage.ImageIntegration_Google) *v1.ImageIntegration_Google {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Google)
	p2.Google = ConvertStorageGoogleConfigToV1GoogleConfig(p1.Google)
	return p2
}

// ConvertSliceStorageImageIntegrationIbmToV1ImageIntegrationIbm converts a slice of *storage.ImageIntegration_Ibm to a slice of *v1.ImageIntegration_Ibm
func ConvertSliceStorageImageIntegrationIbmToV1ImageIntegrationIbm(p1 []*storage.ImageIntegration_Ibm) []*v1.ImageIntegration_Ibm {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Ibm, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationIbmToV1ImageIntegrationIbm(v))
	}
	return p2
}

// ConvertStorageImageIntegrationIbmToV1ImageIntegrationIbm converts from *storage.ImageIntegration_Ibm to *v1.ImageIntegration_Ibm
func ConvertStorageImageIntegrationIbmToV1ImageIntegrationIbm(p1 *storage.ImageIntegration_Ibm) *v1.ImageIntegration_Ibm {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Ibm)
	p2.Ibm = ConvertStorageIBMRegistryConfigToV1IBMRegistryConfig(p1.Ibm)
	return p2
}

// ConvertSliceStorageImageIntegrationQuayToV1ImageIntegrationQuay converts a slice of *storage.ImageIntegration_Quay to a slice of *v1.ImageIntegration_Quay
func ConvertSliceStorageImageIntegrationQuayToV1ImageIntegrationQuay(p1 []*storage.ImageIntegration_Quay) []*v1.ImageIntegration_Quay {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Quay, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationQuayToV1ImageIntegrationQuay(v))
	}
	return p2
}

// ConvertStorageImageIntegrationQuayToV1ImageIntegrationQuay converts from *storage.ImageIntegration_Quay to *v1.ImageIntegration_Quay
func ConvertStorageImageIntegrationQuayToV1ImageIntegrationQuay(p1 *storage.ImageIntegration_Quay) *v1.ImageIntegration_Quay {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Quay)
	p2.Quay = ConvertStorageQuayConfigToV1QuayConfig(p1.Quay)
	return p2
}

// ConvertSliceStorageImageIntegrationSourceToV1ImageIntegrationSource converts a slice of *storage.ImageIntegration_Source to a slice of *v1.ImageIntegration_Source
func ConvertSliceStorageImageIntegrationSourceToV1ImageIntegrationSource(p1 []*storage.ImageIntegration_Source) []*v1.ImageIntegration_Source {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration_Source, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationSourceToV1ImageIntegrationSource(v))
	}
	return p2
}

// ConvertStorageImageIntegrationSourceToV1ImageIntegrationSource converts from *storage.ImageIntegration_Source to *v1.ImageIntegration_Source
func ConvertStorageImageIntegrationSourceToV1ImageIntegrationSource(p1 *storage.ImageIntegration_Source) *v1.ImageIntegration_Source {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration_Source)
	p2.ClusterId = p1.ClusterId
	p2.Namespace = p1.Namespace
	p2.ImagePullSecretName = p1.ImagePullSecretName
	return p2
}

// ConvertSliceStorageClairConfigToV1ClairConfig converts a slice of *storage.ClairConfig to a slice of *v1.ClairConfig
func ConvertSliceStorageClairConfigToV1ClairConfig(p1 []*storage.ClairConfig) []*v1.ClairConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ClairConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageClairConfigToV1ClairConfig(v))
	}
	return p2
}

// ConvertStorageClairConfigToV1ClairConfig converts from *storage.ClairConfig to *v1.ClairConfig
func ConvertStorageClairConfigToV1ClairConfig(p1 *storage.ClairConfig) *v1.ClairConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ClairConfig)
	p2.Endpoint = p1.Endpoint
	p2.Insecure = p1.Insecure
	return p2
}

// ConvertSliceStorageClairV4ConfigToV1ClairV4Config converts a slice of *storage.ClairV4Config to a slice of *v1.ClairV4Config
func ConvertSliceStorageClairV4ConfigToV1ClairV4Config(p1 []*storage.ClairV4Config) []*v1.ClairV4Config {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ClairV4Config, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageClairV4ConfigToV1ClairV4Config(v))
	}
	return p2
}

// ConvertStorageClairV4ConfigToV1ClairV4Config converts from *storage.ClairV4Config to *v1.ClairV4Config
func ConvertStorageClairV4ConfigToV1ClairV4Config(p1 *storage.ClairV4Config) *v1.ClairV4Config {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ClairV4Config)
	p2.Endpoint = p1.Endpoint
	p2.Insecure = p1.Insecure
	return p2
}

// ConvertSliceStorageClairifyConfigToV1ClairifyConfig converts a slice of *storage.ClairifyConfig to a slice of *v1.ClairifyConfig
func ConvertSliceStorageClairifyConfigToV1ClairifyConfig(p1 []*storage.ClairifyConfig) []*v1.ClairifyConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ClairifyConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageClairifyConfigToV1ClairifyConfig(v))
	}
	return p2
}

// ConvertStorageClairifyConfigToV1ClairifyConfig converts from *storage.ClairifyConfig to *v1.ClairifyConfig
func ConvertStorageClairifyConfigToV1ClairifyConfig(p1 *storage.ClairifyConfig) *v1.ClairifyConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ClairifyConfig)
	p2.Endpoint = p1.Endpoint
	p2.GrpcEndpoint = p1.GrpcEndpoint
	p2.NumConcurrentScans = p1.NumConcurrentScans
	return p2
}

// ConvertSliceStorageDockerConfigToV1DockerConfig converts a slice of *storage.DockerConfig to a slice of *v1.DockerConfig
func ConvertSliceStorageDockerConfigToV1DockerConfig(p1 []*storage.DockerConfig) []*v1.DockerConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.DockerConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageDockerConfigToV1DockerConfig(v))
	}
	return p2
}

// ConvertStorageDockerConfigToV1DockerConfig converts from *storage.DockerConfig to *v1.DockerConfig
func ConvertStorageDockerConfigToV1DockerConfig(p1 *storage.DockerConfig) *v1.DockerConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.DockerConfig)
	p2.Endpoint = p1.Endpoint
	p2.Username = p1.Username
	p2.Password = p1.Password
	p2.Insecure = p1.Insecure
	return p2
}

// ConvertSliceStorageECRConfigToV1ECRConfig converts a slice of *storage.ECRConfig to a slice of *v1.ECRConfig
func ConvertSliceStorageECRConfigToV1ECRConfig(p1 []*storage.ECRConfig) []*v1.ECRConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ECRConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageECRConfigToV1ECRConfig(v))
	}
	return p2
}

// ConvertStorageECRConfigToV1ECRConfig converts from *storage.ECRConfig to *v1.ECRConfig
func ConvertStorageECRConfigToV1ECRConfig(p1 *storage.ECRConfig) *v1.ECRConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ECRConfig)
	p2.RegistryId = p1.RegistryId
	p2.AccessKeyId = p1.AccessKeyId
	p2.SecretAccessKey = p1.SecretAccessKey
	p2.Region = p1.Region
	p2.UseIam = p1.UseIam
	p2.Endpoint = p1.Endpoint
	p2.UseAssumeRole = p1.UseAssumeRole
	p2.AssumeRoleId = p1.AssumeRoleId
	p2.AssumeRoleExternalId = p1.AssumeRoleExternalId
	p2.AuthorizationData = ConvertStorageECRConfigAuthorizationDataToV1ECRConfigAuthorizationData(p1.AuthorizationData)
	return p2
}

// ConvertSliceStorageGoogleConfigToV1GoogleConfig converts a slice of *storage.GoogleConfig to a slice of *v1.GoogleConfig
func ConvertSliceStorageGoogleConfigToV1GoogleConfig(p1 []*storage.GoogleConfig) []*v1.GoogleConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.GoogleConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageGoogleConfigToV1GoogleConfig(v))
	}
	return p2
}

// ConvertStorageGoogleConfigToV1GoogleConfig converts from *storage.GoogleConfig to *v1.GoogleConfig
func ConvertStorageGoogleConfigToV1GoogleConfig(p1 *storage.GoogleConfig) *v1.GoogleConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.GoogleConfig)
	p2.Endpoint = p1.Endpoint
	p2.ServiceAccount = p1.ServiceAccount
	p2.Project = p1.Project
	return p2
}

// ConvertSliceStorageIBMRegistryConfigToV1IBMRegistryConfig converts a slice of *storage.IBMRegistryConfig to a slice of *v1.IBMRegistryConfig
func ConvertSliceStorageIBMRegistryConfigToV1IBMRegistryConfig(p1 []*storage.IBMRegistryConfig) []*v1.IBMRegistryConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.IBMRegistryConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageIBMRegistryConfigToV1IBMRegistryConfig(v))
	}
	return p2
}

// ConvertStorageIBMRegistryConfigToV1IBMRegistryConfig converts from *storage.IBMRegistryConfig to *v1.IBMRegistryConfig
func ConvertStorageIBMRegistryConfigToV1IBMRegistryConfig(p1 *storage.IBMRegistryConfig) *v1.IBMRegistryConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.IBMRegistryConfig)
	p2.Endpoint = p1.Endpoint
	p2.ApiKey = p1.ApiKey
	return p2
}

// ConvertSliceStorageQuayConfigToV1QuayConfig converts a slice of *storage.QuayConfig to a slice of *v1.QuayConfig
func ConvertSliceStorageQuayConfigToV1QuayConfig(p1 []*storage.QuayConfig) []*v1.QuayConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.QuayConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageQuayConfigToV1QuayConfig(v))
	}
	return p2
}

// ConvertStorageQuayConfigToV1QuayConfig converts from *storage.QuayConfig to *v1.QuayConfig
func ConvertStorageQuayConfigToV1QuayConfig(p1 *storage.QuayConfig) *v1.QuayConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.QuayConfig)
	p2.Endpoint = p1.Endpoint
	p2.OauthToken = p1.OauthToken
	p2.Insecure = p1.Insecure
	p2.RegistryRobotCredentials = ConvertStorageQuayConfigRobotAccountToV1QuayConfigRobotAccount(p1.RegistryRobotCredentials)
	return p2
}

// ConvertSliceStorageECRConfigAuthorizationDataToV1ECRConfigAuthorizationData converts a slice of *storage.ECRConfig_AuthorizationData to a slice of *v1.ECRConfig_AuthorizationData
func ConvertSliceStorageECRConfigAuthorizationDataToV1ECRConfigAuthorizationData(p1 []*storage.ECRConfig_AuthorizationData) []*v1.ECRConfig_AuthorizationData {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ECRConfig_AuthorizationData, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageECRConfigAuthorizationDataToV1ECRConfigAuthorizationData(v))
	}
	return p2
}

// ConvertStorageECRConfigAuthorizationDataToV1ECRConfigAuthorizationData converts from *storage.ECRConfig_AuthorizationData to *v1.ECRConfig_AuthorizationData
func ConvertStorageECRConfigAuthorizationDataToV1ECRConfigAuthorizationData(p1 *storage.ECRConfig_AuthorizationData) *v1.ECRConfig_AuthorizationData {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ECRConfig_AuthorizationData)
	p2.Username = p1.Username
	p2.Password = p1.Password
	p2.ExpiresAt = p1.ExpiresAt.Clone()
	return p2
}

// ConvertSliceStorageQuayConfigRobotAccountToV1QuayConfigRobotAccount converts a slice of *storage.QuayConfig_RobotAccount to a slice of *v1.QuayConfig_RobotAccount
func ConvertSliceStorageQuayConfigRobotAccountToV1QuayConfigRobotAccount(p1 []*storage.QuayConfig_RobotAccount) []*v1.QuayConfig_RobotAccount {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.QuayConfig_RobotAccount, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageQuayConfigRobotAccountToV1QuayConfigRobotAccount(v))
	}
	return p2
}

// ConvertStorageQuayConfigRobotAccountToV1QuayConfigRobotAccount converts from *storage.QuayConfig_RobotAccount to *v1.QuayConfig_RobotAccount
func ConvertStorageQuayConfigRobotAccountToV1QuayConfigRobotAccount(p1 *storage.QuayConfig_RobotAccount) *v1.QuayConfig_RobotAccount {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.QuayConfig_RobotAccount)
	p2.Username = p1.Username
	p2.Password = p1.Password
	return p2
}

// ConvertSliceStorageImageIntegrationToV1ImageIntegration converts a slice of *storage.ImageIntegration to a slice of *v1.ImageIntegration
func ConvertSliceStorageImageIntegrationToV1ImageIntegration(p1 []*storage.ImageIntegration) []*v1.ImageIntegration {
	if p1 == nil {
		return nil
	}
	p2 := make([]*v1.ImageIntegration, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertStorageImageIntegrationToV1ImageIntegration(v))
	}
	return p2
}

// ConvertStorageImageIntegrationToV1ImageIntegration converts from *storage.ImageIntegration to *v1.ImageIntegration
func ConvertStorageImageIntegrationToV1ImageIntegration(p1 *storage.ImageIntegration) *v1.ImageIntegration {
	if p1 == nil {
		return nil
	}
	p2 := new(v1.ImageIntegration)
	p2.Id = p1.Id
	p2.Name = p1.Name
	p2.Type = p1.Type
	if p1.Categories != nil {
		p2.Categories = make([]v1.ImageIntegrationCategory, len(p1.Categories))
		for idx := range p1.Categories {
			p2.Categories[idx] = v1.ImageIntegrationCategory(p1.Categories[idx])
		}
	}
	if p1.IntegrationConfig != nil {
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Clairify); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationClairifyToV1ImageIntegrationClairify(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Docker); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationDockerToV1ImageIntegrationDocker(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Quay); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationQuayToV1ImageIntegrationQuay(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Ecr); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationEcrToV1ImageIntegrationEcr(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Google); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationGoogleToV1ImageIntegrationGoogle(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Clair); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationClairToV1ImageIntegrationClair(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_ClairV4); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationClairV4ToV1ImageIntegrationClairV4(val)
		}
		if val, ok := p1.IntegrationConfig.(*storage.ImageIntegration_Ibm); ok {
			p2.IntegrationConfig = ConvertStorageImageIntegrationIbmToV1ImageIntegrationIbm(val)
		}
	}
	p2.Autogenerated = p1.Autogenerated
	p2.ClusterId = p1.ClusterId
	p2.SkipTestIntegration = p1.SkipTestIntegration
	p2.Source = ConvertStorageImageIntegrationSourceToV1ImageIntegrationSource(p1.Source)
	return p2
}

// ConvertSliceV1ImageIntegrationClairToStorageImageIntegrationClair converts a slice of *v1.ImageIntegration_Clair to a slice of *storage.ImageIntegration_Clair
func ConvertSliceV1ImageIntegrationClairToStorageImageIntegrationClair(p1 []*v1.ImageIntegration_Clair) []*storage.ImageIntegration_Clair {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Clair, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationClairToStorageImageIntegrationClair(v))
	}
	return p2
}

// ConvertV1ImageIntegrationClairToStorageImageIntegrationClair converts from *v1.ImageIntegration_Clair to *storage.ImageIntegration_Clair
func ConvertV1ImageIntegrationClairToStorageImageIntegrationClair(p1 *v1.ImageIntegration_Clair) *storage.ImageIntegration_Clair {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Clair)
	p2.Clair = ConvertV1ClairConfigToStorageClairConfig(p1.Clair)
	return p2
}

// ConvertSliceV1ImageIntegrationClairV4ToStorageImageIntegrationClairV4 converts a slice of *v1.ImageIntegration_ClairV4 to a slice of *storage.ImageIntegration_ClairV4
func ConvertSliceV1ImageIntegrationClairV4ToStorageImageIntegrationClairV4(p1 []*v1.ImageIntegration_ClairV4) []*storage.ImageIntegration_ClairV4 {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_ClairV4, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationClairV4ToStorageImageIntegrationClairV4(v))
	}
	return p2
}

// ConvertV1ImageIntegrationClairV4ToStorageImageIntegrationClairV4 converts from *v1.ImageIntegration_ClairV4 to *storage.ImageIntegration_ClairV4
func ConvertV1ImageIntegrationClairV4ToStorageImageIntegrationClairV4(p1 *v1.ImageIntegration_ClairV4) *storage.ImageIntegration_ClairV4 {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_ClairV4)
	p2.ClairV4 = ConvertV1ClairV4ConfigToStorageClairV4Config(p1.ClairV4)
	return p2
}

// ConvertSliceV1ImageIntegrationClairifyToStorageImageIntegrationClairify converts a slice of *v1.ImageIntegration_Clairify to a slice of *storage.ImageIntegration_Clairify
func ConvertSliceV1ImageIntegrationClairifyToStorageImageIntegrationClairify(p1 []*v1.ImageIntegration_Clairify) []*storage.ImageIntegration_Clairify {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Clairify, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationClairifyToStorageImageIntegrationClairify(v))
	}
	return p2
}

// ConvertV1ImageIntegrationClairifyToStorageImageIntegrationClairify converts from *v1.ImageIntegration_Clairify to *storage.ImageIntegration_Clairify
func ConvertV1ImageIntegrationClairifyToStorageImageIntegrationClairify(p1 *v1.ImageIntegration_Clairify) *storage.ImageIntegration_Clairify {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Clairify)
	p2.Clairify = ConvertV1ClairifyConfigToStorageClairifyConfig(p1.Clairify)
	return p2
}

// ConvertSliceV1ImageIntegrationDockerToStorageImageIntegrationDocker converts a slice of *v1.ImageIntegration_Docker to a slice of *storage.ImageIntegration_Docker
func ConvertSliceV1ImageIntegrationDockerToStorageImageIntegrationDocker(p1 []*v1.ImageIntegration_Docker) []*storage.ImageIntegration_Docker {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Docker, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationDockerToStorageImageIntegrationDocker(v))
	}
	return p2
}

// ConvertV1ImageIntegrationDockerToStorageImageIntegrationDocker converts from *v1.ImageIntegration_Docker to *storage.ImageIntegration_Docker
func ConvertV1ImageIntegrationDockerToStorageImageIntegrationDocker(p1 *v1.ImageIntegration_Docker) *storage.ImageIntegration_Docker {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Docker)
	p2.Docker = ConvertV1DockerConfigToStorageDockerConfig(p1.Docker)
	return p2
}

// ConvertSliceV1ImageIntegrationEcrToStorageImageIntegrationEcr converts a slice of *v1.ImageIntegration_Ecr to a slice of *storage.ImageIntegration_Ecr
func ConvertSliceV1ImageIntegrationEcrToStorageImageIntegrationEcr(p1 []*v1.ImageIntegration_Ecr) []*storage.ImageIntegration_Ecr {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Ecr, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationEcrToStorageImageIntegrationEcr(v))
	}
	return p2
}

// ConvertV1ImageIntegrationEcrToStorageImageIntegrationEcr converts from *v1.ImageIntegration_Ecr to *storage.ImageIntegration_Ecr
func ConvertV1ImageIntegrationEcrToStorageImageIntegrationEcr(p1 *v1.ImageIntegration_Ecr) *storage.ImageIntegration_Ecr {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Ecr)
	p2.Ecr = ConvertV1ECRConfigToStorageECRConfig(p1.Ecr)
	return p2
}

// ConvertSliceV1ImageIntegrationGoogleToStorageImageIntegrationGoogle converts a slice of *v1.ImageIntegration_Google to a slice of *storage.ImageIntegration_Google
func ConvertSliceV1ImageIntegrationGoogleToStorageImageIntegrationGoogle(p1 []*v1.ImageIntegration_Google) []*storage.ImageIntegration_Google {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Google, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationGoogleToStorageImageIntegrationGoogle(v))
	}
	return p2
}

// ConvertV1ImageIntegrationGoogleToStorageImageIntegrationGoogle converts from *v1.ImageIntegration_Google to *storage.ImageIntegration_Google
func ConvertV1ImageIntegrationGoogleToStorageImageIntegrationGoogle(p1 *v1.ImageIntegration_Google) *storage.ImageIntegration_Google {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Google)
	p2.Google = ConvertV1GoogleConfigToStorageGoogleConfig(p1.Google)
	return p2
}

// ConvertSliceV1ImageIntegrationIbmToStorageImageIntegrationIbm converts a slice of *v1.ImageIntegration_Ibm to a slice of *storage.ImageIntegration_Ibm
func ConvertSliceV1ImageIntegrationIbmToStorageImageIntegrationIbm(p1 []*v1.ImageIntegration_Ibm) []*storage.ImageIntegration_Ibm {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Ibm, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationIbmToStorageImageIntegrationIbm(v))
	}
	return p2
}

// ConvertV1ImageIntegrationIbmToStorageImageIntegrationIbm converts from *v1.ImageIntegration_Ibm to *storage.ImageIntegration_Ibm
func ConvertV1ImageIntegrationIbmToStorageImageIntegrationIbm(p1 *v1.ImageIntegration_Ibm) *storage.ImageIntegration_Ibm {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Ibm)
	p2.Ibm = ConvertV1IBMRegistryConfigToStorageIBMRegistryConfig(p1.Ibm)
	return p2
}

// ConvertSliceV1ImageIntegrationQuayToStorageImageIntegrationQuay converts a slice of *v1.ImageIntegration_Quay to a slice of *storage.ImageIntegration_Quay
func ConvertSliceV1ImageIntegrationQuayToStorageImageIntegrationQuay(p1 []*v1.ImageIntegration_Quay) []*storage.ImageIntegration_Quay {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Quay, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationQuayToStorageImageIntegrationQuay(v))
	}
	return p2
}

// ConvertV1ImageIntegrationQuayToStorageImageIntegrationQuay converts from *v1.ImageIntegration_Quay to *storage.ImageIntegration_Quay
func ConvertV1ImageIntegrationQuayToStorageImageIntegrationQuay(p1 *v1.ImageIntegration_Quay) *storage.ImageIntegration_Quay {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Quay)
	p2.Quay = ConvertV1QuayConfigToStorageQuayConfig(p1.Quay)
	return p2
}

// ConvertSliceV1ImageIntegrationSourceToStorageImageIntegrationSource converts a slice of *v1.ImageIntegration_Source to a slice of *storage.ImageIntegration_Source
func ConvertSliceV1ImageIntegrationSourceToStorageImageIntegrationSource(p1 []*v1.ImageIntegration_Source) []*storage.ImageIntegration_Source {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration_Source, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationSourceToStorageImageIntegrationSource(v))
	}
	return p2
}

// ConvertV1ImageIntegrationSourceToStorageImageIntegrationSource converts from *v1.ImageIntegration_Source to *storage.ImageIntegration_Source
func ConvertV1ImageIntegrationSourceToStorageImageIntegrationSource(p1 *v1.ImageIntegration_Source) *storage.ImageIntegration_Source {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration_Source)
	p2.ClusterId = p1.ClusterId
	p2.Namespace = p1.Namespace
	p2.ImagePullSecretName = p1.ImagePullSecretName
	return p2
}

// ConvertSliceV1ClairConfigToStorageClairConfig converts a slice of *v1.ClairConfig to a slice of *storage.ClairConfig
func ConvertSliceV1ClairConfigToStorageClairConfig(p1 []*v1.ClairConfig) []*storage.ClairConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ClairConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ClairConfigToStorageClairConfig(v))
	}
	return p2
}

// ConvertV1ClairConfigToStorageClairConfig converts from *v1.ClairConfig to *storage.ClairConfig
func ConvertV1ClairConfigToStorageClairConfig(p1 *v1.ClairConfig) *storage.ClairConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ClairConfig)
	p2.Endpoint = p1.Endpoint
	p2.Insecure = p1.Insecure
	return p2
}

// ConvertSliceV1ClairV4ConfigToStorageClairV4Config converts a slice of *v1.ClairV4Config to a slice of *storage.ClairV4Config
func ConvertSliceV1ClairV4ConfigToStorageClairV4Config(p1 []*v1.ClairV4Config) []*storage.ClairV4Config {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ClairV4Config, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ClairV4ConfigToStorageClairV4Config(v))
	}
	return p2
}

// ConvertV1ClairV4ConfigToStorageClairV4Config converts from *v1.ClairV4Config to *storage.ClairV4Config
func ConvertV1ClairV4ConfigToStorageClairV4Config(p1 *v1.ClairV4Config) *storage.ClairV4Config {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ClairV4Config)
	p2.Endpoint = p1.Endpoint
	p2.Insecure = p1.Insecure
	return p2
}

// ConvertSliceV1ClairifyConfigToStorageClairifyConfig converts a slice of *v1.ClairifyConfig to a slice of *storage.ClairifyConfig
func ConvertSliceV1ClairifyConfigToStorageClairifyConfig(p1 []*v1.ClairifyConfig) []*storage.ClairifyConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ClairifyConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ClairifyConfigToStorageClairifyConfig(v))
	}
	return p2
}

// ConvertV1ClairifyConfigToStorageClairifyConfig converts from *v1.ClairifyConfig to *storage.ClairifyConfig
func ConvertV1ClairifyConfigToStorageClairifyConfig(p1 *v1.ClairifyConfig) *storage.ClairifyConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ClairifyConfig)
	p2.Endpoint = p1.Endpoint
	p2.GrpcEndpoint = p1.GrpcEndpoint
	p2.NumConcurrentScans = p1.NumConcurrentScans
	return p2
}

// ConvertSliceV1DockerConfigToStorageDockerConfig converts a slice of *v1.DockerConfig to a slice of *storage.DockerConfig
func ConvertSliceV1DockerConfigToStorageDockerConfig(p1 []*v1.DockerConfig) []*storage.DockerConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.DockerConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1DockerConfigToStorageDockerConfig(v))
	}
	return p2
}

// ConvertV1DockerConfigToStorageDockerConfig converts from *v1.DockerConfig to *storage.DockerConfig
func ConvertV1DockerConfigToStorageDockerConfig(p1 *v1.DockerConfig) *storage.DockerConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.DockerConfig)
	p2.Endpoint = p1.Endpoint
	p2.Username = p1.Username
	p2.Password = p1.Password
	p2.Insecure = p1.Insecure
	return p2
}

// ConvertSliceV1ECRConfigToStorageECRConfig converts a slice of *v1.ECRConfig to a slice of *storage.ECRConfig
func ConvertSliceV1ECRConfigToStorageECRConfig(p1 []*v1.ECRConfig) []*storage.ECRConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ECRConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ECRConfigToStorageECRConfig(v))
	}
	return p2
}

// ConvertV1ECRConfigToStorageECRConfig converts from *v1.ECRConfig to *storage.ECRConfig
func ConvertV1ECRConfigToStorageECRConfig(p1 *v1.ECRConfig) *storage.ECRConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ECRConfig)
	p2.RegistryId = p1.RegistryId
	p2.AccessKeyId = p1.AccessKeyId
	p2.SecretAccessKey = p1.SecretAccessKey
	p2.Region = p1.Region
	p2.UseIam = p1.UseIam
	p2.Endpoint = p1.Endpoint
	p2.UseAssumeRole = p1.UseAssumeRole
	p2.AssumeRoleId = p1.AssumeRoleId
	p2.AssumeRoleExternalId = p1.AssumeRoleExternalId
	p2.AuthorizationData = ConvertV1ECRConfigAuthorizationDataToStorageECRConfigAuthorizationData(p1.AuthorizationData)
	return p2
}

// ConvertSliceV1GoogleConfigToStorageGoogleConfig converts a slice of *v1.GoogleConfig to a slice of *storage.GoogleConfig
func ConvertSliceV1GoogleConfigToStorageGoogleConfig(p1 []*v1.GoogleConfig) []*storage.GoogleConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.GoogleConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1GoogleConfigToStorageGoogleConfig(v))
	}
	return p2
}

// ConvertV1GoogleConfigToStorageGoogleConfig converts from *v1.GoogleConfig to *storage.GoogleConfig
func ConvertV1GoogleConfigToStorageGoogleConfig(p1 *v1.GoogleConfig) *storage.GoogleConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.GoogleConfig)
	p2.Endpoint = p1.Endpoint
	p2.ServiceAccount = p1.ServiceAccount
	p2.Project = p1.Project
	return p2
}

// ConvertSliceV1IBMRegistryConfigToStorageIBMRegistryConfig converts a slice of *v1.IBMRegistryConfig to a slice of *storage.IBMRegistryConfig
func ConvertSliceV1IBMRegistryConfigToStorageIBMRegistryConfig(p1 []*v1.IBMRegistryConfig) []*storage.IBMRegistryConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.IBMRegistryConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1IBMRegistryConfigToStorageIBMRegistryConfig(v))
	}
	return p2
}

// ConvertV1IBMRegistryConfigToStorageIBMRegistryConfig converts from *v1.IBMRegistryConfig to *storage.IBMRegistryConfig
func ConvertV1IBMRegistryConfigToStorageIBMRegistryConfig(p1 *v1.IBMRegistryConfig) *storage.IBMRegistryConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.IBMRegistryConfig)
	p2.Endpoint = p1.Endpoint
	p2.ApiKey = p1.ApiKey
	return p2
}

// ConvertSliceV1QuayConfigToStorageQuayConfig converts a slice of *v1.QuayConfig to a slice of *storage.QuayConfig
func ConvertSliceV1QuayConfigToStorageQuayConfig(p1 []*v1.QuayConfig) []*storage.QuayConfig {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.QuayConfig, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1QuayConfigToStorageQuayConfig(v))
	}
	return p2
}

// ConvertV1QuayConfigToStorageQuayConfig converts from *v1.QuayConfig to *storage.QuayConfig
func ConvertV1QuayConfigToStorageQuayConfig(p1 *v1.QuayConfig) *storage.QuayConfig {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.QuayConfig)
	p2.Endpoint = p1.Endpoint
	p2.OauthToken = p1.OauthToken
	p2.Insecure = p1.Insecure
	p2.RegistryRobotCredentials = ConvertV1QuayConfigRobotAccountToStorageQuayConfigRobotAccount(p1.RegistryRobotCredentials)
	return p2
}

// ConvertSliceV1ECRConfigAuthorizationDataToStorageECRConfigAuthorizationData converts a slice of *v1.ECRConfig_AuthorizationData to a slice of *storage.ECRConfig_AuthorizationData
func ConvertSliceV1ECRConfigAuthorizationDataToStorageECRConfigAuthorizationData(p1 []*v1.ECRConfig_AuthorizationData) []*storage.ECRConfig_AuthorizationData {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ECRConfig_AuthorizationData, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ECRConfigAuthorizationDataToStorageECRConfigAuthorizationData(v))
	}
	return p2
}

// ConvertV1ECRConfigAuthorizationDataToStorageECRConfigAuthorizationData converts from *v1.ECRConfig_AuthorizationData to *storage.ECRConfig_AuthorizationData
func ConvertV1ECRConfigAuthorizationDataToStorageECRConfigAuthorizationData(p1 *v1.ECRConfig_AuthorizationData) *storage.ECRConfig_AuthorizationData {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ECRConfig_AuthorizationData)
	p2.Username = p1.Username
	p2.Password = p1.Password
	p2.ExpiresAt = p1.ExpiresAt.Clone()
	return p2
}

// ConvertSliceV1QuayConfigRobotAccountToStorageQuayConfigRobotAccount converts a slice of *v1.QuayConfig_RobotAccount to a slice of *storage.QuayConfig_RobotAccount
func ConvertSliceV1QuayConfigRobotAccountToStorageQuayConfigRobotAccount(p1 []*v1.QuayConfig_RobotAccount) []*storage.QuayConfig_RobotAccount {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.QuayConfig_RobotAccount, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1QuayConfigRobotAccountToStorageQuayConfigRobotAccount(v))
	}
	return p2
}

// ConvertV1QuayConfigRobotAccountToStorageQuayConfigRobotAccount converts from *v1.QuayConfig_RobotAccount to *storage.QuayConfig_RobotAccount
func ConvertV1QuayConfigRobotAccountToStorageQuayConfigRobotAccount(p1 *v1.QuayConfig_RobotAccount) *storage.QuayConfig_RobotAccount {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.QuayConfig_RobotAccount)
	p2.Username = p1.Username
	p2.Password = p1.Password
	return p2
}

// ConvertSliceV1ImageIntegrationToStorageImageIntegration converts a slice of *v1.ImageIntegration to a slice of *storage.ImageIntegration
func ConvertSliceV1ImageIntegrationToStorageImageIntegration(p1 []*v1.ImageIntegration) []*storage.ImageIntegration {
	if p1 == nil {
		return nil
	}
	p2 := make([]*storage.ImageIntegration, 0, len(p1))
	for _, v := range p1 {
		p2 = append(p2, ConvertV1ImageIntegrationToStorageImageIntegration(v))
	}
	return p2
}

// ConvertV1ImageIntegrationToStorageImageIntegration converts from *v1.ImageIntegration to *storage.ImageIntegration
func ConvertV1ImageIntegrationToStorageImageIntegration(p1 *v1.ImageIntegration) *storage.ImageIntegration {
	if p1 == nil {
		return nil
	}
	p2 := new(storage.ImageIntegration)
	p2.Id = p1.Id
	p2.Name = p1.Name
	p2.Type = p1.Type
	if p1.Categories != nil {
		p2.Categories = make([]storage.ImageIntegrationCategory, len(p1.Categories))
		for idx := range p1.Categories {
			p2.Categories[idx] = storage.ImageIntegrationCategory(p1.Categories[idx])
		}
	}
	if p1.IntegrationConfig != nil {
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Clairify); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationClairifyToStorageImageIntegrationClairify(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Docker); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationDockerToStorageImageIntegrationDocker(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Quay); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationQuayToStorageImageIntegrationQuay(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Ecr); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationEcrToStorageImageIntegrationEcr(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Google); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationGoogleToStorageImageIntegrationGoogle(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Clair); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationClairToStorageImageIntegrationClair(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_ClairV4); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationClairV4ToStorageImageIntegrationClairV4(val)
		}
		if val, ok := p1.IntegrationConfig.(*v1.ImageIntegration_Ibm); ok {
			p2.IntegrationConfig = ConvertV1ImageIntegrationIbmToStorageImageIntegrationIbm(val)
		}
	}
	p2.Autogenerated = p1.Autogenerated
	p2.ClusterId = p1.ClusterId
	p2.SkipTestIntegration = p1.SkipTestIntegration
	p2.Source = ConvertV1ImageIntegrationSourceToStorageImageIntegrationSource(p1.Source)
	return p2
}

