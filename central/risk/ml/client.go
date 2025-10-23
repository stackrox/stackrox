package ml

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// MLRiskClient provides interface to ML Risk Service
type MLRiskClient interface {
	GetDeploymentRisk(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*MLRiskResponse, error)
	GetBatchDeploymentRisk(ctx context.Context, requests []*DeploymentRiskRequest) ([]*MLRiskResponse, error)
	TrainModel(ctx context.Context, trainingData []*TrainingExample) (*TrainingResponse, error)
	GetModelHealth(ctx context.Context) (*ModelHealthResponse, error)
	Close() error
}

// Config for ML Risk Service client
type Config struct {
	Endpoint   string        `yaml:"endpoint"`
	TLSEnabled bool          `yaml:"tls_enabled"`
	TLSConfig  *tls.Config   `yaml:"-"`
	Timeout    time.Duration `yaml:"timeout"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Endpoint:   "ml-risk-service:8080",
		TLSEnabled: false,
		Timeout:    30 * time.Second,
	}
}

// MLRiskResponse represents the response from ML service
type MLRiskResponse struct {
	DeploymentID       string               `json:"deployment_id"`
	RiskScore          float32              `json:"risk_score"`
	FeatureImportances []*FeatureImportance `json:"feature_importances"`
	ModelVersion       string               `json:"model_version"`
	Timestamp          int64                `json:"timestamp"`
}

// FeatureImportance represents importance of a single feature
type FeatureImportance struct {
	FeatureName     string  `json:"feature_name"`
	ImportanceScore float32 `json:"importance_score"`
	FeatureCategory string  `json:"feature_category"`
	Description     string  `json:"description"`
}

// DeploymentRiskRequest represents a request for deployment risk assessment
type DeploymentRiskRequest struct {
	DeploymentID       string              `json:"deployment_id"`
	DeploymentFeatures *DeploymentFeatures `json:"deployment_features"`
	ImageFeatures      []*ImageFeatures    `json:"image_features"`
}

// DeploymentFeatures represents deployment-level features
type DeploymentFeatures struct {
	PolicyViolationCount          int32   `json:"policy_violation_count"`
	PolicyViolationSeverityScore  float32 `json:"policy_violation_severity_score"`
	ProcessBaselineViolations     int32   `json:"process_baseline_violations"`
	HostNetwork                   bool    `json:"host_network"`
	HostPID                       bool    `json:"host_pid"`
	HostIPC                       bool    `json:"host_ipc"`
	PrivilegedContainerCount      int32   `json:"privileged_container_count"`
	AutomountServiceAccountToken  bool    `json:"automount_service_account_token"`
	ExposedPortCount              int32   `json:"exposed_port_count"`
	HasExternalExposure           bool    `json:"has_external_exposure"`
	ServiceAccountPermissionLevel float32 `json:"service_account_permission_level"`
	ReplicaCount                  int32   `json:"replica_count"`
	IsOrchestratorComponent       bool    `json:"is_orchestrator_component"`
	IsPlatformComponent           bool    `json:"is_platform_component"`
	ClusterID                     string  `json:"cluster_id"`
	Namespace                     string  `json:"namespace"`
	CreationTimestamp             int64   `json:"creation_timestamp"`
	IsInactive                    bool    `json:"is_inactive"`
}

// ImageFeatures represents image-level features
type ImageFeatures struct {
	ImageID                string  `json:"image_id"`
	ImageName              string  `json:"image_name"`
	CriticalVulnCount      int32   `json:"critical_vuln_count"`
	HighVulnCount          int32   `json:"high_vuln_count"`
	MediumVulnCount        int32   `json:"medium_vuln_count"`
	LowVulnCount           int32   `json:"low_vuln_count"`
	AvgCVSSScore           float32 `json:"avg_cvss_score"`
	MaxCVSSScore           float32 `json:"max_cvss_score"`
	TotalComponentCount    int32   `json:"total_component_count"`
	RiskyComponentCount    int32   `json:"risky_component_count"`
	ImageCreationTimestamp int64   `json:"image_creation_timestamp"`
	ImageAgeDays           int32   `json:"image_age_days"`
	IsClusterLocal         bool    `json:"is_cluster_local"`
	BaseImage              string  `json:"base_image"`
	LayerCount             int32   `json:"layer_count"`
}

// TrainingExample represents a training example
type TrainingExample struct {
	DeploymentFeatures *DeploymentFeatures `json:"deployment_features"`
	ImageFeatures      []*ImageFeatures    `json:"image_features"`
	CurrentRiskScore   float32             `json:"current_risk_score"`
	DeploymentID       string              `json:"deployment_id"`
}

// TrainingResponse represents training response
type TrainingResponse struct {
	Success      bool             `json:"success"`
	ModelVersion string           `json:"model_version"`
	Metrics      *TrainingMetrics `json:"metrics"`
	ErrorMessage string           `json:"error_message"`
}

// TrainingMetrics represents training metrics
type TrainingMetrics struct {
	ValidationNDCG          float32              `json:"validation_ndcg"`
	ValidationAUC           float32              `json:"validation_auc"`
	TrainingLoss            float32              `json:"training_loss"`
	EpochsCompleted         int32                `json:"epochs_completed"`
	GlobalFeatureImportance []*FeatureImportance `json:"global_feature_importance"`
}

// ModelHealthResponse represents model health response
type ModelHealthResponse struct {
	Healthy               bool          `json:"healthy"`
	ModelVersion          string        `json:"model_version"`
	LastTrainingTime      int64         `json:"last_training_time"`
	TrainingExamplesCount int32         `json:"training_examples_count"`
	CurrentMetrics        *ModelMetrics `json:"current_metrics"`
}

// ModelMetrics represents current model metrics
type ModelMetrics struct {
	CurrentNDCG         float32 `json:"current_ndcg"`
	CurrentAUC          float32 `json:"current_auc"`
	PredictionsServed   int32   `json:"predictions_served"`
	AvgPredictionTimeMs float32 `json:"avg_prediction_time_ms"`
}

// mlRiskClientImpl implements MLRiskClient
type mlRiskClientImpl struct {
	config *Config
	conn   *grpc.ClientConn
	// Note: In practice, this would use generated gRPC client stub
	// client ml_risk_pb.MLRiskServiceClient
}

// NewMLRiskClient creates a new ML Risk Service client
func NewMLRiskClient(config *Config) (MLRiskClient, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var opts []grpc.DialOption

	// Configure TLS
	if config.TLSEnabled {
		if config.TLSConfig != nil {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(config.TLSConfig)))
		} else {
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Add timeout
	opts = append(opts, grpc.WithTimeout(config.Timeout))

	// Dial the ML service
	conn, err := grpc.Dial(config.Endpoint, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ML Risk Service at %s: %w", config.Endpoint, err)
	}

	client := &mlRiskClientImpl{
		config: config,
		conn:   conn,
		// client: ml_risk_pb.NewMLRiskServiceClient(conn),
	}

	log.Infof("Connected to ML Risk Service at %s", config.Endpoint)
	return client, nil
}

// GetDeploymentRisk gets risk score for a single deployment
func (c *mlRiskClientImpl) GetDeploymentRisk(ctx context.Context, deployment *storage.Deployment, images []*storage.Image) (*MLRiskResponse, error) {
	// Extract features from StackRox objects (for future gRPC implementation)
	// deploymentFeatures := c.extractDeploymentFeatures(deployment)
	// imageFeatures := c.extractImageFeatures(images)

	// In practice, this would call the gRPC method:
	// request := &DeploymentRiskRequest{
	//     DeploymentID:       deployment.GetId(),
	//     DeploymentFeatures: deploymentFeatures,
	//     ImageFeatures:      imageFeatures,
	// }
	// resp, err := c.client.GetDeploymentRisk(ctx, convertToProtoRequest(request))

	// For now, return a mock response
	return &MLRiskResponse{
		DeploymentID: deployment.GetId(),
		RiskScore:    2.5, // Mock risk score
		ModelVersion: "mock-v1.0",
		Timestamp:    time.Now().Unix(),
		FeatureImportances: []*FeatureImportance{
			{
				FeatureName:     "policy_violations",
				ImportanceScore: 0.35,
				FeatureCategory: "policy",
				Description:     "Policy violation severity score",
			},
			{
				FeatureName:     "vulnerabilities",
				ImportanceScore: 0.28,
				FeatureCategory: "security",
				Description:     "Image vulnerability score",
			},
		},
	}, nil
}

// GetBatchDeploymentRisk gets risk scores for multiple deployments
func (c *mlRiskClientImpl) GetBatchDeploymentRisk(ctx context.Context, requests []*DeploymentRiskRequest) ([]*MLRiskResponse, error) {
	responses := make([]*MLRiskResponse, len(requests))

	// In practice, this would make a single batch gRPC call
	// For now, simulate batch processing
	for i, req := range requests {
		responses[i] = &MLRiskResponse{
			DeploymentID: req.DeploymentID,
			RiskScore:    2.0 + float32(i%5)*0.5, // Mock varying scores
			ModelVersion: "mock-v1.0",
			Timestamp:    time.Now().Unix(),
		}
	}

	return responses, nil
}

// TrainModel triggers model training
func (c *mlRiskClientImpl) TrainModel(ctx context.Context, trainingData []*TrainingExample) (*TrainingResponse, error) {
	// In practice, this would call the gRPC training method
	return &TrainingResponse{
		Success:      true,
		ModelVersion: "v1.1",
		Metrics: &TrainingMetrics{
			ValidationNDCG:  0.85,
			ValidationAUC:   0.78,
			TrainingLoss:    0.32,
			EpochsCompleted: 50,
		},
	}, nil
}

// GetModelHealth checks model health
func (c *mlRiskClientImpl) GetModelHealth(ctx context.Context) (*ModelHealthResponse, error) {
	// In practice, this would call the gRPC health method
	return &ModelHealthResponse{
		Healthy:               true,
		ModelVersion:          "v1.0",
		LastTrainingTime:      time.Now().Add(-24 * time.Hour).Unix(),
		TrainingExamplesCount: 5000,
		CurrentMetrics: &ModelMetrics{
			CurrentNDCG:         0.83,
			CurrentAUC:          0.76,
			PredictionsServed:   1523,
			AvgPredictionTimeMs: 45.2,
		},
	}, nil
}

// Close closes the client connection
func (c *mlRiskClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// extractDeploymentFeatures converts StackRox Deployment to DeploymentFeatures
func (c *mlRiskClientImpl) extractDeploymentFeatures(deployment *storage.Deployment) *DeploymentFeatures {
	features := &DeploymentFeatures{
		ReplicaCount:                 int32(deployment.GetReplicas()),
		HostNetwork:                  deployment.GetHostNetwork(),
		HostPID:                      deployment.GetHostPid(),
		HostIPC:                      deployment.GetHostIpc(),
		AutomountServiceAccountToken: deployment.GetAutomountServiceAccountToken(),
		IsOrchestratorComponent:      deployment.GetOrchestratorComponent(),
		IsPlatformComponent:          deployment.GetPlatformComponent(),
		ClusterID:                    deployment.GetClusterId(),
		Namespace:                    deployment.GetNamespace(),
		IsInactive:                   deployment.GetInactive(),
	}

	// Extract creation timestamp
	if created := deployment.GetCreated(); created != nil {
		features.CreationTimestamp = created.GetSeconds()
	}

	// Extract service account permission level
	features.ServiceAccountPermissionLevel = float32(deployment.GetServiceAccountPermissionLevel())

	// Count privileged containers
	privilegedCount := int32(0)
	for _, container := range deployment.GetContainers() {
		if container.GetSecurityContext().GetPrivileged() {
			privilegedCount++
		}
	}
	features.PrivilegedContainerCount = privilegedCount

	// Count exposed ports
	features.ExposedPortCount = int32(len(deployment.GetPorts()))

	// Check for external exposure
	for _, port := range deployment.GetPorts() {
		if port.GetExposure() == storage.PortConfig_EXTERNAL {
			features.HasExternalExposure = true
			break
		}
	}

	return features
}

// extractImageFeatures converts StackRox Images to ImageFeatures
func (c *mlRiskClientImpl) extractImageFeatures(images []*storage.Image) []*ImageFeatures {
	features := make([]*ImageFeatures, len(images))

	for i, image := range images {
		imageFeatures := &ImageFeatures{
			ImageID:        image.GetId(),
			ImageName:      image.GetName().GetFullName(),
			IsClusterLocal: image.GetIsClusterLocal(),
		}

		// Extract creation timestamp
		if metadata := image.GetMetadata(); metadata != nil {
			if created := metadata.GetV1().GetCreated(); created != nil {
				imageFeatures.ImageCreationTimestamp = created.GetSeconds()

				// Calculate age in days
				ageDays := (time.Now().Unix() - created.GetSeconds()) / (24 * 3600)
				imageFeatures.ImageAgeDays = int32(ageDays)
			}
			imageFeatures.LayerCount = int32(len(metadata.GetLayerShas()))
		}

		// Extract vulnerability information
		if scan := image.GetScan(); scan != nil {
			components := scan.GetComponents()
			imageFeatures.TotalComponentCount = int32(len(components))

			var cvssScores []float32
			riskyComponents := int32(0)

			for _, component := range components {
				hasHighRiskVuln := false
				for _, vuln := range component.GetVulns() {
					// Count vulnerabilities by severity
					switch vuln.GetSeverity() {
					case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
						imageFeatures.CriticalVulnCount++
						hasHighRiskVuln = true
					case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
						imageFeatures.HighVulnCount++
						hasHighRiskVuln = true
					case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
						imageFeatures.MediumVulnCount++
					case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
						imageFeatures.LowVulnCount++
					}

					// Collect CVSS scores
					if cvss := vuln.GetCvss(); cvss > 0 {
						cvssScores = append(cvssScores, cvss)
					}
				}

				if hasHighRiskVuln {
					riskyComponents++
				}
			}

			imageFeatures.RiskyComponentCount = riskyComponents

			// Calculate CVSS statistics
			if len(cvssScores) > 0 {
				sum := float32(0)
				max := float32(0)
				for _, score := range cvssScores {
					sum += score
					if score > max {
						max = score
					}
				}
				imageFeatures.AvgCVSSScore = sum / float32(len(cvssScores))
				imageFeatures.MaxCVSSScore = max
			}
		}

		features[i] = imageFeatures
	}

	return features
}
