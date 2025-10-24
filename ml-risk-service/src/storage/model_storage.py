"""
Model storage abstraction layer for ML Risk Service.
Provides unified interface for storing and retrieving trained models with support for
multiple storage backends including local filesystem, cloud storage, and distributed storage.
"""

import os
import logging
import json
import hashlib
import shutil
from abc import ABC, abstractmethod
from datetime import datetime, timezone
from typing import Dict, Any, List, Optional, Union, Tuple
from dataclasses import dataclass, asdict
from pathlib import Path
import tempfile

# Storage backends
try:
    from google.cloud import storage
    GCS_AVAILABLE = True
except ImportError:
    GCS_AVAILABLE = False


logger = logging.getLogger(__name__)


@dataclass
class ModelMetadata:
    """Enhanced model metadata for tracking and versioning."""
    model_id: str
    version: str
    algorithm: str
    feature_count: int
    training_timestamp: str
    model_size_bytes: int
    checksum: str
    performance_metrics: Dict[str, float]
    config: Dict[str, Any]
    tags: Dict[str, str] = None
    description: str = ""
    created_by: str = "ml-risk-service"

    # Enhanced versioning fields
    semantic_version: Optional[str] = None  # e.g., "1.2.3"
    parent_version: Optional[str] = None  # For tracking lineage
    status: str = "draft"  # draft, staging, production, deprecated
    deployment_stage: str = "development"  # development, testing, staging, production

    # Model lineage and comparison
    baseline_model_id: Optional[str] = None
    baseline_version: Optional[str] = None
    performance_comparison: Optional[Dict[str, float]] = None  # vs baseline

    # Validation and testing
    validation_metrics: Optional[Dict[str, float]] = None
    test_dataset_id: Optional[str] = None
    test_dataset_size: int = 0

    # Deployment tracking
    first_deployed_at: Optional[str] = None
    last_deployed_at: Optional[str] = None
    deployment_count: int = 0

    # Quality metrics
    model_quality_score: Optional[float] = None
    drift_score: Optional[float] = None
    stability_score: Optional[float] = None

    # Metadata management
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
    archived_at: Optional[str] = None

    def __post_init__(self):
        """Post-initialization to set default values."""
        if self.created_at is None:
            self.created_at = datetime.now(timezone.utc).isoformat()
        self.updated_at = datetime.now(timezone.utc).isoformat()

        # Generate semantic version if not provided
        if self.semantic_version is None:
            self.semantic_version = self._generate_semantic_version()

    def _generate_semantic_version(self) -> str:
        """Generate semantic version from regular version."""
        if self.version.startswith('v'):
            version_num = self.version[1:]
        else:
            version_num = self.version

        # Try to parse as semantic version
        try:
            parts = version_num.split('.')
            if len(parts) == 1:
                return f"{parts[0]}.0.0"
            elif len(parts) == 2:
                return f"{parts[0]}.{parts[1]}.0"
            else:
                return version_num
        except:
            # Fallback for non-numeric versions
            return f"1.{hash(self.version) % 1000}.0"

    def update_status(self, new_status: str, deployment_stage: str = None):
        """Update model status with proper validation."""
        valid_statuses = ["draft", "staging", "production", "deprecated", "archived"]
        if new_status not in valid_statuses:
            raise ValueError(f"Invalid status: {new_status}. Must be one of {valid_statuses}")

        self.status = new_status
        if deployment_stage:
            self.deployment_stage = deployment_stage
        self.updated_at = datetime.now(timezone.utc).isoformat()

        # Track deployment
        if new_status == "production":
            if self.first_deployed_at is None:
                self.first_deployed_at = self.updated_at
            self.last_deployed_at = self.updated_at
            self.deployment_count += 1

    def compare_performance(self, baseline_metrics: Dict[str, float]) -> Dict[str, float]:
        """Compare performance against baseline."""
        comparison = {}
        for metric, value in self.performance_metrics.items():
            if metric in baseline_metrics:
                baseline_value = baseline_metrics[metric]
                if baseline_value != 0:
                    improvement = ((value - baseline_value) / baseline_value) * 100
                    comparison[f"{metric}_improvement_pct"] = improvement
                comparison[f"{metric}_baseline"] = baseline_value
                comparison[f"{metric}_current"] = value

        self.performance_comparison = comparison
        return comparison

    def calculate_quality_score(self) -> float:
        """Calculate overall model quality score."""
        if not self.performance_metrics:
            return 0.0

        # Weight different metrics (can be configured)
        weights = {
            "validation_ndcg": 0.4,
            "validation_auc": 0.3,
            "training_loss": -0.2,  # Negative because lower is better
            "stability_score": 0.1
        }

        score = 0.0
        total_weight = 0.0

        for metric, value in self.performance_metrics.items():
            if metric in weights:
                weight = weights[metric]
                if metric == "training_loss":
                    # Invert loss (lower is better)
                    normalized_value = max(0, 1 - value)
                else:
                    normalized_value = min(1.0, max(0.0, value))

                score += weight * normalized_value
                total_weight += abs(weight)

        if total_weight > 0:
            self.model_quality_score = score / total_weight
        else:
            self.model_quality_score = 0.0

        return self.model_quality_score

    def is_production_ready(self) -> Tuple[bool, List[str]]:
        """Check if model is ready for production deployment."""
        issues = []

        # Check required metrics
        required_metrics = ["validation_ndcg", "validation_auc"]
        for metric in required_metrics:
            if metric not in self.performance_metrics:
                issues.append(f"Missing required metric: {metric}")

        # Check performance thresholds
        if "validation_ndcg" in self.performance_metrics:
            if self.performance_metrics["validation_ndcg"] < 0.7:
                issues.append("NDCG below production threshold (0.7)")

        if "validation_auc" in self.performance_metrics:
            if self.performance_metrics["validation_auc"] < 0.65:
                issues.append("AUC below production threshold (0.65)")

        # Check model quality
        if self.model_quality_score is None:
            self.calculate_quality_score()

        if self.model_quality_score < 0.8:
            issues.append(f"Model quality score too low: {self.model_quality_score:.3f}")

        # Check testing
        if self.test_dataset_size == 0:
            issues.append("No test dataset validation performed")

        return len(issues) == 0, issues

    def get_version_lineage(self) -> List[str]:
        """Get the version lineage chain."""
        lineage = [self.version]
        if self.parent_version:
            lineage.insert(0, self.parent_version)
        return lineage

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary."""
        return asdict(self)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ModelMetadata':
        """Create from dictionary."""
        # Handle None values for optional fields
        for field in ["performance_comparison", "validation_metrics"]:
            if field in data and data[field] is None:
                data[field] = {}

        return cls(**data)


@dataclass
class StorageConfig:
    """Configuration for model storage."""
    backend: str  # 'local', 'gcs'
    base_path: str
    encryption_enabled: bool = False
    compression_enabled: bool = True
    retention_policy: Dict[str, Any] = None
    backup_enabled: bool = True
    backup_frequency: str = "daily"  # 'hourly', 'daily', 'weekly'

    # Cloud-specific settings
    gcs_project_id: Optional[str] = None
    gcs_credentials_path: Optional[str] = None

    @classmethod
    def from_env(cls, backend: str = None) -> 'StorageConfig':
        """Create storage config from environment variables."""
        backend = backend or os.getenv('ROX_ML_MODEL_STORAGE_BACKEND', 'local')

        config = cls(
            backend=backend,
            base_path=os.getenv('ROX_ML_MODEL_STORAGE_BASE_PATH', '/app/models'),
            encryption_enabled=os.getenv('ROX_ML_MODEL_ENCRYPTION_ENABLED', 'false').lower() == 'true',
            compression_enabled=os.getenv('ROX_ML_MODEL_COMPRESSION_ENABLED', 'true').lower() == 'true',
            backup_enabled=os.getenv('ROX_ML_MODEL_BACKUP_ENABLED', 'false').lower() == 'true',
            backup_frequency=os.getenv('ROX_ML_MODEL_BACKUP_FREQUENCY', 'daily'),
        )

        # Add cloud-specific configurations
        if backend == 'gcs':
            config.gcs_project_id = os.getenv('ROX_ML_GCS_PROJECT_ID')
            config.gcs_credentials_path = os.getenv('ROX_ML_GCS_CREDENTIALS_PATH')
            # For GCS, base_path should include bucket name
            if not config.base_path.startswith('gs://'):
                bucket_name = os.getenv('ROX_ML_GCS_BUCKET_NAME', 'stackrox-ml-models')
                config.base_path = f"{bucket_name}/models"

        return config


class ModelStorage(ABC):
    """Abstract base class for model storage backends."""

    def __init__(self, config: StorageConfig):
        self.config = config
        self.logger = logging.getLogger(f"{__name__}.{self.__class__.__name__}")

    @abstractmethod
    def save_model(self, model_data: bytes, metadata: ModelMetadata) -> bool:
        """Save model data with metadata."""
        pass

    @abstractmethod
    def load_model(self, model_id: str, version: Optional[str] = None) -> Tuple[bytes, ModelMetadata]:
        """Load model data and metadata."""
        pass

    @abstractmethod
    def list_models(self, model_id: Optional[str] = None) -> List[ModelMetadata]:
        """List available models."""
        pass

    @abstractmethod
    def delete_model(self, model_id: str, version: Optional[str] = None) -> bool:
        """Delete model."""
        pass

    @abstractmethod
    def model_exists(self, model_id: str, version: Optional[str] = None) -> bool:
        """Check if model exists."""
        pass

    def _calculate_checksum(self, data: bytes) -> str:
        """Calculate SHA256 checksum of model data."""
        return hashlib.sha256(data).hexdigest()

    def _get_model_path(self, model_id: str, version: str) -> str:
        """Get storage path for model."""
        return f"models/{model_id}/v{version}"

    def _get_metadata_path(self, model_id: str, version: str) -> str:
        """Get storage path for metadata."""
        return f"models/{model_id}/v{version}/metadata.json"


class LocalModelStorage(ModelStorage):
    """Local filesystem storage backend."""

    def __init__(self, config: StorageConfig):
        super().__init__(config)
        self.base_path = Path(config.base_path)
        self.base_path.mkdir(parents=True, exist_ok=True)

    def save_model(self, model_data: bytes, metadata: ModelMetadata) -> bool:
        """Save model to local filesystem."""
        try:
            model_path = self.base_path / self._get_model_path(metadata.model_id, metadata.version)
            model_path.mkdir(parents=True, exist_ok=True)

            # Save model data
            model_file = model_path / "model.joblib"
            with open(model_file, 'wb') as f:
                f.write(model_data)

            # Update metadata with file info
            metadata.model_size_bytes = len(model_data)
            metadata.checksum = self._calculate_checksum(model_data)

            # Save metadata
            metadata_file = model_path / "metadata.json"
            with open(metadata_file, 'w') as f:
                json.dump(metadata.to_dict(), f, indent=2)

            self.logger.info(f"Model {metadata.model_id} v{metadata.version} saved to {model_path}")
            return True

        except Exception as e:
            self.logger.error(f"Failed to save model {metadata.model_id}: {e}")
            return False

    def load_model(self, model_id: str, version: Optional[str] = None) -> Tuple[bytes, ModelMetadata]:
        """Load model from local filesystem."""
        if version is None:
            version = self._get_latest_version(model_id)
            if not version:
                raise FileNotFoundError(f"No versions found for model {model_id}")

        model_path = self.base_path / self._get_model_path(model_id, version)

        if not model_path.exists():
            raise FileNotFoundError(f"Model {model_id} v{version} not found")

        # Load metadata
        metadata_file = model_path / "metadata.json"
        with open(metadata_file, 'r') as f:
            metadata_dict = json.load(f)
        metadata = ModelMetadata.from_dict(metadata_dict)

        # Load model data
        model_file = model_path / "model.joblib"
        with open(model_file, 'rb') as f:
            model_data = f.read()

        # Verify checksum
        if metadata.checksum != self._calculate_checksum(model_data):
            raise ValueError(f"Checksum mismatch for model {model_id} v{version}")

        self.logger.info(f"Model {model_id} v{version} loaded from {model_path}")
        return model_data, metadata

    def list_models(self, model_id: Optional[str] = None) -> List[ModelMetadata]:
        """List available models."""
        models = []
        models_dir = self.base_path / "models"

        if not models_dir.exists():
            return models

        if model_id:
            # List versions for specific model
            model_dir = models_dir / model_id
            if model_dir.exists():
                for version_dir in model_dir.iterdir():
                    if version_dir.is_dir() and version_dir.name.startswith('v'):
                        metadata_file = version_dir / "metadata.json"
                        if metadata_file.exists():
                            try:
                                with open(metadata_file, 'r') as f:
                                    metadata_dict = json.load(f)
                                models.append(ModelMetadata.from_dict(metadata_dict))
                            except Exception as e:
                                self.logger.warning(f"Failed to load metadata from {metadata_file}: {e}")
        else:
            # List all models
            for model_dir in models_dir.iterdir():
                if model_dir.is_dir():
                    models.extend(self.list_models(model_dir.name))

        return sorted(models, key=lambda m: m.training_timestamp, reverse=True)

    def delete_model(self, model_id: str, version: Optional[str] = None) -> bool:
        """Delete model from local filesystem."""
        try:
            if version:
                # Delete specific version
                model_path = self.base_path / self._get_model_path(model_id, version)
                if model_path.exists():
                    shutil.rmtree(model_path)
                    self.logger.info(f"Deleted model {model_id} v{version}")
                    return True
            else:
                # Delete all versions
                model_dir = self.base_path / "models" / model_id
                if model_dir.exists():
                    shutil.rmtree(model_dir)
                    self.logger.info(f"Deleted all versions of model {model_id}")
                    return True

            return False

        except Exception as e:
            self.logger.error(f"Failed to delete model {model_id}: {e}")
            return False

    def model_exists(self, model_id: str, version: Optional[str] = None) -> bool:
        """Check if model exists."""
        if version is None:
            model_dir = self.base_path / "models" / model_id
            return model_dir.exists() and any(model_dir.iterdir())
        else:
            model_path = self.base_path / self._get_model_path(model_id, version)
            return model_path.exists() and (model_path / "model.joblib").exists()

    def _get_latest_version(self, model_id: str) -> Optional[str]:
        """Get the latest version of a model."""
        model_dir = self.base_path / "models" / model_id
        if not model_dir.exists():
            return None

        versions = []
        for version_dir in model_dir.iterdir():
            if version_dir.is_dir() and version_dir.name.startswith('v'):
                version_num = version_dir.name[1:]  # Remove 'v' prefix
                try:
                    versions.append(int(version_num))
                except ValueError:
                    # Handle non-numeric versions
                    versions.append(version_num)

        if not versions:
            return None

        # Return highest numeric version or latest string version
        if all(isinstance(v, int) for v in versions):
            return str(max(versions))
        else:
            return str(sorted(versions)[-1])










class GCSModelStorage(ModelStorage):
    """Google Cloud Storage backend."""

    def __init__(self, config: StorageConfig):
        super().__init__(config)

        if not GCS_AVAILABLE:
            raise ImportError("google-cloud-storage is required for GCS storage")

        self.bucket_name = config.base_path.split('/')[-1]
        self.prefix = '/'.join(config.base_path.split('/')[:-1]).strip('/')

        try:
            # Initialize GCS client
            if hasattr(config, 'gcs_credentials_path') and config.gcs_credentials_path:
                self.client = storage.Client.from_service_account_json(config.gcs_credentials_path)
            else:
                # Use default credentials (e.g., from environment)
                self.client = storage.Client()

            # Get bucket reference
            self.bucket = self.client.bucket(self.bucket_name)

            # Test connection
            self.bucket.reload()

        except Exception as e:
            raise ConnectionError(f"Failed to connect to GCS: {e}")

    def save_model(self, model_data: bytes, metadata: ModelMetadata) -> bool:
        """Save model to GCS."""
        try:
            # Update metadata
            metadata.model_size_bytes = len(model_data)
            metadata.checksum = self._calculate_checksum(model_data)

            # Upload model data
            model_path = f"{self.prefix}/{self._get_model_path(metadata.model_id, metadata.version)}/model.joblib"
            model_blob = self.bucket.blob(model_path)
            model_blob.metadata = {'checksum': metadata.checksum}
            model_blob.upload_from_string(model_data, content_type='application/octet-stream')

            # Upload metadata
            metadata_path = f"{self.prefix}/{self._get_metadata_path(metadata.model_id, metadata.version)}"
            metadata_blob = self.bucket.blob(metadata_path)
            metadata_blob.upload_from_string(
                json.dumps(metadata.to_dict(), indent=2),
                content_type='application/json'
            )

            self.logger.info(f"Model {metadata.model_id} v{metadata.version} saved to GCS")
            return True

        except Exception as e:
            self.logger.error(f"Failed to save model to GCS: {e}")
            return False

    def load_model(self, model_id: str, version: Optional[str] = None) -> Tuple[bytes, ModelMetadata]:
        """Load model from GCS."""
        if version is None:
            version = self._get_latest_version(model_id)
            if not version:
                raise FileNotFoundError(f"No versions found for model {model_id}")

        try:
            # Load metadata
            metadata_path = f"{self.prefix}/{self._get_metadata_path(model_id, version)}"
            metadata_blob = self.bucket.blob(metadata_path)

            if not metadata_blob.exists():
                raise FileNotFoundError(f"Model {model_id} v{version} not found in GCS")

            metadata_content = metadata_blob.download_as_text()
            metadata_dict = json.loads(metadata_content)
            metadata = ModelMetadata.from_dict(metadata_dict)

            # Load model data
            model_path = f"{self.prefix}/{self._get_model_path(model_id, version)}/model.joblib"
            model_blob = self.bucket.blob(model_path)
            model_data = model_blob.download_as_bytes()

            # Verify checksum
            if metadata.checksum != self._calculate_checksum(model_data):
                raise ValueError(f"Checksum mismatch for model {model_id} v{version}")

            self.logger.info(f"Model {model_id} v{version} loaded from GCS")
            return model_data, metadata

        except Exception as e:
            if "not found" in str(e).lower():
                raise FileNotFoundError(f"Model {model_id} v{version} not found in GCS")
            raise

    def list_models(self, model_id: Optional[str] = None) -> List[ModelMetadata]:
        """List available models in GCS."""
        models = []

        try:
            prefix = f"{self.prefix}/models/"
            if model_id:
                prefix += f"{model_id}/"

            # List all metadata files
            for blob in self.bucket.list_blobs(prefix=prefix):
                if blob.name.endswith('metadata.json'):
                    try:
                        metadata_content = blob.download_as_text()
                        metadata_dict = json.loads(metadata_content)
                        models.append(ModelMetadata.from_dict(metadata_dict))
                    except Exception as e:
                        self.logger.warning(f"Failed to load metadata from {blob.name}: {e}")

        except Exception as e:
            self.logger.error(f"Failed to list models from GCS: {e}")

        return sorted(models, key=lambda m: m.training_timestamp, reverse=True)

    def delete_model(self, model_id: str, version: Optional[str] = None) -> bool:
        """Delete model from GCS."""
        try:
            prefix = f"{self.prefix}/models/{model_id}/"
            if version:
                prefix += f"v{version}/"

            # List and delete all objects with the prefix
            blobs_to_delete = list(self.bucket.list_blobs(prefix=prefix))

            for blob in blobs_to_delete:
                blob.delete()

            self.logger.info(f"Deleted model {model_id} v{version} from GCS")
            return True

        except Exception as e:
            self.logger.error(f"Failed to delete model from GCS: {e}")
            return False

    def model_exists(self, model_id: str, version: Optional[str] = None) -> bool:
        """Check if model exists in GCS."""
        try:
            if version:
                metadata_path = f"{self.prefix}/{self._get_metadata_path(model_id, version)}"
                blob = self.bucket.blob(metadata_path)
                return blob.exists()
            else:
                prefix = f"{self.prefix}/models/{model_id}/"
                # Check if any blobs exist with this prefix
                for blob in self.bucket.list_blobs(prefix=prefix, max_results=1):
                    return True
                return False

        except Exception:
            return False

    def _get_latest_version(self, model_id: str) -> Optional[str]:
        """Get the latest version of a model from GCS."""
        try:
            prefix = f"{self.prefix}/models/{model_id}/"
            versions = []

            # List all version directories
            for blob in self.bucket.list_blobs(prefix=prefix, delimiter='/'):
                # We need to look at the prefixes to find version directories
                pass

            # Use a different approach: list all metadata files and extract versions
            for blob in self.bucket.list_blobs(prefix=prefix):
                if blob.name.endswith('metadata.json'):
                    path_parts = blob.name.split('/')
                    for part in path_parts:
                        if part.startswith('v'):
                            version_num = part[1:]
                            try:
                                versions.append(int(version_num))
                            except ValueError:
                                versions.append(version_num)
                            break

            if not versions:
                return None

            if all(isinstance(v, int) for v in versions):
                return str(max(versions))
            else:
                return str(sorted(versions)[-1])

        except Exception as e:
            self.logger.error(f"Failed to get latest version for {model_id}: {e}")
            return None











class ModelStorageManager:
    """High-level model storage manager with backup and recovery."""

    def __init__(self, config: StorageConfig):
        self.config = config
        self.logger = logging.getLogger(__name__)

        # Initialize primary storage
        self.primary_storage = self._create_storage_backend(config)

        # Initialize backup storage if enabled
        self.backup_storage = None
        if config.backup_enabled and hasattr(config, 'backup_config'):
            try:
                self.backup_storage = self._create_storage_backend(config.backup_config)
            except Exception as e:
                self.logger.warning(f"Failed to initialize backup storage: {e}")

    def _create_storage_backend(self, config: StorageConfig) -> ModelStorage:
        """Create storage backend based on configuration."""
        if config.backend == 'local':
            return LocalModelStorage(config)
        elif config.backend == 'gcs':
            return GCSModelStorage(config)
        else:
            raise ValueError(f"Unknown storage backend: {config.backend} (supported: local, gcs)")

    def save_model(self, model_data: bytes, metadata: ModelMetadata) -> bool:
        """Save model with automatic backup."""
        # Save to primary storage
        success = self.primary_storage.save_model(model_data, metadata)

        if success and self.backup_storage:
            # Save to backup storage
            try:
                self.backup_storage.save_model(model_data, metadata)
                self.logger.info(f"Model {metadata.model_id} backed up successfully")
            except Exception as e:
                self.logger.error(f"Failed to backup model {metadata.model_id}: {e}")

        return success

    def load_model(self, model_id: str, version: Optional[str] = None) -> Tuple[bytes, ModelMetadata]:
        """Load model with fallback to backup."""
        try:
            return self.primary_storage.load_model(model_id, version)
        except Exception as e:
            self.logger.warning(f"Failed to load from primary storage: {e}")

            if self.backup_storage:
                self.logger.info("Attempting to load from backup storage")
                return self.backup_storage.load_model(model_id, version)

            raise

    def list_models(self, model_id: Optional[str] = None) -> List[ModelMetadata]:
        """List models from primary storage."""
        return self.primary_storage.list_models(model_id)

    def delete_model(self, model_id: str, version: Optional[str] = None) -> bool:
        """Delete model from both primary and backup storage."""
        success = self.primary_storage.delete_model(model_id, version)

        if self.backup_storage:
            try:
                self.backup_storage.delete_model(model_id, version)
            except Exception as e:
                self.logger.warning(f"Failed to delete from backup storage: {e}")

        return success

    def model_exists(self, model_id: str, version: Optional[str] = None) -> bool:
        """Check if model exists in primary storage."""
        return self.primary_storage.model_exists(model_id, version)

    def verify_model_integrity(self, model_id: str, version: Optional[str] = None) -> bool:
        """Verify model integrity by comparing primary and backup."""
        if not self.backup_storage:
            return True  # No backup to compare

        try:
            primary_data, primary_metadata = self.primary_storage.load_model(model_id, version)
            backup_data, backup_metadata = self.backup_storage.load_model(model_id, version)

            return (primary_metadata.checksum == backup_metadata.checksum and
                    self.primary_storage._calculate_checksum(primary_data) == primary_metadata.checksum)

        except Exception as e:
            self.logger.error(f"Failed to verify model integrity: {e}")
            return False

    def get_storage_stats(self) -> Dict[str, Any]:
        """Get storage statistics."""
        models = self.list_models()
        total_size = sum(m.model_size_bytes for m in models)

        return {
            'total_models': len(models),
            'total_size_bytes': total_size,
            'storage_backend': self.config.backend,
            'backup_enabled': self.backup_storage is not None,
            'latest_model': models[0].model_id if models else None
        }

    # Enhanced versioning and metadata tracking methods

    def create_model_version(self, model_data: bytes, metadata: ModelMetadata,
                           parent_version: str = None) -> bool:
        """Create a new model version with lineage tracking."""
        # Set parent version for lineage
        if parent_version:
            metadata.parent_version = parent_version

        # Generate next semantic version if not provided
        if not metadata.semantic_version:
            metadata.semantic_version = self._generate_next_semantic_version(
                metadata.model_id, parent_version
            )

        # Calculate quality score
        metadata.calculate_quality_score()

        # Save the model
        success = self.save_model(model_data, metadata)

        if success:
            self.logger.info(f"Created model version {metadata.model_id} v{metadata.version} "
                           f"(semantic: {metadata.semantic_version})")

        return success

    def promote_model_version(self, model_id: str, version: str,
                            new_status: str, deployment_stage: str = None) -> bool:
        """Promote a model version to a new status."""
        try:
            # Load current metadata
            _, metadata = self.primary_storage.load_model(model_id, version)

            # Update status
            old_status = metadata.status
            metadata.update_status(new_status, deployment_stage)

            # Re-save metadata
            model_data, _ = self.primary_storage.load_model(model_id, version)
            success = self.primary_storage.save_model(model_data, metadata)

            if success:
                self.logger.info(f"Promoted model {model_id} v{version} from {old_status} to {new_status}")

            return success

        except Exception as e:
            self.logger.error(f"Failed to promote model {model_id} v{version}: {e}")
            return False

    def compare_model_versions(self, model_id: str, version1: str, version2: str) -> Dict[str, Any]:
        """Compare two model versions."""
        try:
            _, metadata1 = self.primary_storage.load_model(model_id, version1)
            _, metadata2 = self.primary_storage.load_model(model_id, version2)

            comparison = {
                'model_id': model_id,
                'version1': {
                    'version': version1,
                    'semantic_version': metadata1.semantic_version,
                    'status': metadata1.status,
                    'performance_metrics': metadata1.performance_metrics,
                    'quality_score': metadata1.model_quality_score,
                    'created_at': metadata1.created_at
                },
                'version2': {
                    'version': version2,
                    'semantic_version': metadata2.semantic_version,
                    'status': metadata2.status,
                    'performance_metrics': metadata2.performance_metrics,
                    'quality_score': metadata2.model_quality_score,
                    'created_at': metadata2.created_at
                },
                'performance_diff': {},
                'quality_diff': None
            }

            # Calculate performance differences
            for metric in metadata1.performance_metrics:
                if metric in metadata2.performance_metrics:
                    diff = metadata2.performance_metrics[metric] - metadata1.performance_metrics[metric]
                    comparison['performance_diff'][metric] = diff

            # Calculate quality difference
            if metadata1.model_quality_score and metadata2.model_quality_score:
                comparison['quality_diff'] = metadata2.model_quality_score - metadata1.model_quality_score

            return comparison

        except Exception as e:
            self.logger.error(f"Failed to compare model versions: {e}")
            return {}

    def get_model_lineage(self, model_id: str, version: str) -> List[Dict[str, Any]]:
        """Get the complete lineage of a model version."""
        lineage = []
        current_version = version

        try:
            while current_version:
                _, metadata = self.primary_storage.load_model(model_id, current_version)
                lineage.append({
                    'version': current_version,
                    'semantic_version': metadata.semantic_version,
                    'status': metadata.status,
                    'created_at': metadata.created_at,
                    'performance_metrics': metadata.performance_metrics,
                    'quality_score': metadata.model_quality_score
                })
                current_version = metadata.parent_version

        except Exception as e:
            self.logger.error(f"Failed to build lineage for {model_id} v{version}: {e}")

        return lineage

    def get_production_models(self) -> List[ModelMetadata]:
        """Get all models currently in production."""
        all_models = self.list_models()
        return [m for m in all_models if m.status == "production"]

    def get_models_by_status(self, status: str) -> List[ModelMetadata]:
        """Get all models with a specific status."""
        all_models = self.list_models()
        return [m for m in all_models if m.status == status]

    def archive_old_versions(self, model_id: str, keep_count: int = 5) -> int:
        """Archive old versions, keeping only the most recent ones."""
        try:
            models = self.list_models(model_id)
            if len(models) <= keep_count:
                return 0

            # Sort by creation time (newest first)
            models.sort(key=lambda m: m.created_at or "", reverse=True)

            archived_count = 0
            for model in models[keep_count:]:
                if model.status != "production":  # Don't archive production models
                    model.status = "archived"
                    model.archived_at = datetime.now(timezone.utc).isoformat()

                    # Re-save metadata
                    model_data, _ = self.primary_storage.load_model(model_id, model.version)
                    self.primary_storage.save_model(model_data, model)
                    archived_count += 1

            self.logger.info(f"Archived {archived_count} old versions of model {model_id}")
            return archived_count

        except Exception as e:
            self.logger.error(f"Failed to archive old versions for {model_id}: {e}")
            return 0

    def validate_model_for_production(self, model_id: str, version: str) -> Tuple[bool, List[str]]:
        """Validate if a model version is ready for production."""
        try:
            _, metadata = self.primary_storage.load_model(model_id, version)
            return metadata.is_production_ready()

        except Exception as e:
            return False, [f"Failed to load model for validation: {e}"]

    def _generate_next_semantic_version(self, model_id: str, parent_version: str = None) -> str:
        """Generate the next semantic version for a model."""
        try:
            if parent_version:
                # Load parent metadata to get semantic version
                _, parent_metadata = self.primary_storage.load_model(model_id, parent_version)
                parent_semantic = parent_metadata.semantic_version or "1.0.0"
            else:
                # Find latest version
                models = self.list_models(model_id)
                if not models:
                    return "1.0.0"
                parent_semantic = models[0].semantic_version or "1.0.0"

            # Parse semantic version and increment patch
            parts = parent_semantic.split('.')
            if len(parts) >= 3:
                major, minor, patch = int(parts[0]), int(parts[1]), int(parts[2])
                return f"{major}.{minor}.{patch + 1}"
            else:
                return "1.0.1"

        except Exception:
            # Fallback to simple versioning
            return "1.0.0"

    def get_model_metrics_history(self, model_id: str, metric_name: str) -> List[Dict[str, Any]]:
        """Get the history of a specific metric across all versions."""
        try:
            models = self.list_models(model_id)
            history = []

            for model in models:
                if metric_name in model.performance_metrics:
                    history.append({
                        'version': model.version,
                        'semantic_version': model.semantic_version,
                        'value': model.performance_metrics[metric_name],
                        'created_at': model.created_at
                    })

            # Sort by creation time
            history.sort(key=lambda x: x['created_at'] or "")
            return history

        except Exception as e:
            self.logger.error(f"Failed to get metrics history for {model_id}: {e}")
            return []