"""
ML ranking model for deployment risk assessment with explainability.
"""

import logging
import pickle
import json
import numpy as np
import pandas as pd
from typing import Dict, Any, List, Tuple, Optional, Union
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
import joblib
import io

# ML libraries
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from sklearn.metrics import ndcg_score, roc_auc_score
from sklearn.ensemble import RandomForestRegressor

# Explainability
try:
    import shap
    SHAP_AVAILABLE = True
except ImportError:
    SHAP_AVAILABLE = False
    logging.warning("SHAP not available - feature importance will be limited")

logger = logging.getLogger(__name__)


@dataclass
class ModelMetrics:
    """Model performance metrics for RandomForest training."""
    train_ndcg: float = 0.0
    val_ndcg: float = 0.0
    feature_importance: Dict[str, float] = None


@dataclass
class PredictionResult:
    """Result of a risk prediction."""
    risk_score: float
    feature_importance: Dict[str, float]
    model_version: str
    confidence: float = 0.0


class RiskRankingModel:
    """
    ML model for deployment risk ranking with explainability.
    Currently supports sklearn-based regression approach with extensible architecture
    for future algorithm implementations.
    """

    def __init__(self, config: Optional[Dict[str, Any]] = None):
        self.config = config or self._default_config()
        self.model = None
        self.scaler = None
        self.feature_names = None
        self.model_version = None
        self.training_metrics = None
        self.shap_explainer = None

        # Initialize model based on configuration
        self.algorithm = self.config.get('model', {}).get('algorithm', 'sklearn_ranksvm')

    def _default_config(self) -> Dict[str, Any]:
        """Default model configuration."""
        return {
            'model': {
                'algorithm': 'sklearn_ranksvm',
                'validation_split': 0.2,
                'random_state': 42,
                'sklearn_params': {
                    'n_estimators': 100,
                    'n_jobs': -1
                },
                'explainability': {
                    'shap_enabled': True,
                    'top_features': 10
                }
            }
        }

    def train(self, X: np.ndarray, y: np.ndarray,
              groups: Optional[np.ndarray] = None,
              feature_names: Optional[List[str]] = None) -> ModelMetrics:
        """
        Train the ranking model.

        Args:
            X: Feature matrix (n_samples, n_features)
            y: Risk scores (n_samples,)
            groups: Group sizes for ranking (optional)
            feature_names: Names of features

        Returns:
            Training metrics
        """
        logger.info(f"Training {self.algorithm} model with {X.shape[0]} samples, {X.shape[1]} features")

        self.feature_names = feature_names or [f"feature_{i}" for i in range(X.shape[1])]

        # Check for sufficient variance in target values
        y_variance = np.var(y)
        y_unique = len(np.unique(y))
        y_min, y_max = np.min(y), np.max(y)
        y_range = y_max - y_min
        logger.info(f"Target variance: {y_variance:.6f}, unique values: {y_unique}, range: [{y_min:.3f}, {y_max:.3f}] (span: {y_range:.3f})")

        # Only return dummy metrics for truly identical targets
        if y_unique == 1:
            logger.error("All target values are identical! Cannot train ranking model.")
            # Return dummy metrics for identical targets
            return ModelMetrics(
                train_ndcg=0.0,
                val_ndcg=0.0,
                feature_importance={name: 0.0 for name in self.feature_names}
            )

        # Log warning for low variance but continue training
        if y_variance < 1e-4:
            logger.warning(f"Low target variance detected (var={y_variance:.6f}, unique={y_unique}). "
                          "Training will continue but model may have limited discriminative power.")

        # For very low variance, still proceed but with enhanced logging
        if y_variance < 1e-6:
            logger.warning(f"Very low variance detected. Will attempt training with available data. "
                          f"Consider reviewing data quality and synthetic scoring.")
            # Log sample of y values for debugging
            sample_size = min(10, len(y))
            y_sample = y[:sample_size]
            logger.info(f"Sample of target values: {y_sample}")


        # Split data
        val_split = self.config.get('model', {}).get('validation_split', 0.2)
        random_state = self.config.get('model', {}).get('random_state', 42)

        # Split data for training and validation
        X_train, X_val, y_train, y_val = train_test_split(
            X, y, test_size=val_split, random_state=random_state)

        # Analyze raw features before scaling
        raw_feature_variances = [np.var(X_train[:, i]) for i in range(X_train.shape[1])]
        raw_zero_variance_count = sum(1 for v in raw_feature_variances if v < 1e-12)
        logger.info(f"Raw feature analysis (before scaling):")
        logger.info(f"  Zero-variance features: {raw_zero_variance_count}/{len(raw_feature_variances)}")
        logger.info(f"  Raw feature variance range: [{min(raw_feature_variances):.10f}, {max(raw_feature_variances):.10f}]")

        # Show sample values for first few features to debug
        for i in range(min(5, X_train.shape[1])):
            feature_name = self.feature_names[i] if i < len(self.feature_names) else f"feature_{i}"
            unique_vals = len(np.unique(X_train[:, i]))
            min_val, max_val = np.min(X_train[:, i]), np.max(X_train[:, i])
            logger.info(f"  {feature_name}: {unique_vals} unique values, range [{min_val:.6f}, {max_val:.6f}], variance: {raw_feature_variances[i]:.10f}")

        # Handle zero-variance features before scaling
        if raw_zero_variance_count > 0:
            logger.warning(f"Found {raw_zero_variance_count} zero-variance features. Applying variance enhancement.")

            # Add small random noise to zero-variance features to enable meaningful scaling
            X_train_enhanced = X_train.copy()
            X_val_enhanced = X_val.copy()

            np.random.seed(42)  # Deterministic noise
            for i in range(X_train.shape[1]):
                if raw_feature_variances[i] < 1e-12:
                    # Add small noise proportional to the mean value (or 0.01 if mean is zero)
                    mean_val = np.mean(X_train[:, i])
                    noise_std = max(0.01, abs(mean_val) * 0.01)  # 1% of mean or 0.01 minimum

                    train_noise = np.random.normal(0, noise_std, X_train.shape[0])
                    val_noise = np.random.normal(0, noise_std, X_val.shape[0])

                    X_train_enhanced[:, i] += train_noise
                    X_val_enhanced[:, i] += val_noise

                    feature_name = self.feature_names[i] if i < len(self.feature_names) else f"feature_{i}"
                    logger.info(f"  Enhanced {feature_name} with noise (std={noise_std:.6f})")

            # Use enhanced features for scaling
            X_train_for_scaling = X_train_enhanced
            X_val_for_scaling = X_val_enhanced
        else:
            X_train_for_scaling = X_train
            X_val_for_scaling = X_val

        # Scale features
        self.scaler = StandardScaler()
        X_train_scaled = self.scaler.fit_transform(X_train_for_scaling)
        X_val_scaled = self.scaler.transform(X_val_for_scaling)

        # Use float scores directly for sklearn
        logger.info(f"Using float scores - Train: {y_train.min():.6f}-{y_train.max():.6f}, Val: {y_val.min():.6f}-{y_val.max():.6f}")

        # Feature variance analysis before training
        feature_variances = [np.var(X_train_scaled[:, i]) for i in range(X_train_scaled.shape[1])]
        total_feature_variance = sum(feature_variances)
        zero_variance_count = sum(1 for v in feature_variances if v < 1e-10)
        logger.info(f"Pre-training feature analysis:")
        logger.info(f"  Total feature variance: {total_feature_variance:.6f}")
        logger.info(f"  Zero-variance features: {zero_variance_count}/{len(feature_variances)}")
        logger.info(f"  Feature variance range: [{min(feature_variances):.6f}, {max(feature_variances):.6f}]")

        if zero_variance_count > len(feature_variances) * 0.5:
            logger.warning(f"More than 50% of features have zero variance! This will cause poor feature importance.")

        # Target variance on training split
        train_target_variance = np.var(y_train)
        train_unique_targets = len(np.unique(y_train))
        logger.info(f"Training split target analysis:")
        logger.info(f"  Target variance: {train_target_variance:.6f}")
        logger.info(f"  Unique target values: {train_unique_targets}")

        if train_target_variance < 1e-6:
            logger.warning(f"Training target variance is extremely low ({train_target_variance:.10f}). RandomForest may not learn meaningful patterns.")

        # Train model
        metrics = self._train_model(X_train_scaled, y_train, X_val_scaled, y_val)

        self.training_metrics = metrics
        self.model_version = f"{self.algorithm}_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        # Initialize SHAP explainer
        if SHAP_AVAILABLE and self.config.get('model', {}).get('explainability', {}).get('shap_enabled', True):
            self._initialize_shap_explainer(X_train_scaled)

        logger.info(f"Training complete. Validation NDCG: {metrics.val_ndcg:.4f}")
        return metrics


    def _train_model(self, X_train: np.ndarray, y_train: np.ndarray,
                     X_val: np.ndarray, y_val: np.ndarray) -> ModelMetrics:
        """Train the machine learning model."""

        # Get algorithm-specific parameters
        sklearn_params = self.config.get('model', {}).get('sklearn_params', {})
        random_state = self.config.get('model', {}).get('random_state', 42)

        # Create model with configurable parameters
        model_params = {
            'random_state': random_state,
            **sklearn_params  # Merge in configured parameters
        }

        self.model = RandomForestRegressor(**model_params)

        self.model.fit(X_train, y_train)

        # Calculate metrics
        train_pred = self.model.predict(X_train)
        val_pred = self.model.predict(X_val)

        train_ndcg = self._calculate_regression_ndcg(y_train, train_pred)
        val_ndcg = self._calculate_regression_ndcg(y_val, val_pred)

        # Feature importance with fallback calculation
        try:
            importance_values = self.model.feature_importances_
            importance_dict = dict(zip(self.feature_names, importance_values))

            # Enhanced diagnostics for feature importance
            total_importance = sum(importance_values)
            max_importance = max(importance_values) if len(importance_values) > 0 else 0.0
            min_importance = min(importance_values) if len(importance_values) > 0 else 0.0
            non_zero_count = sum(1 for v in importance_values if v > 0)

            logger.info(f"RandomForest feature importance analysis:")
            logger.info(f"  Total: {total_importance:.6f}, Max: {max_importance:.6f}, Min: {min_importance:.6f}")
            logger.info(f"  Non-zero features: {non_zero_count}/{len(importance_values)}")
            logger.info(f"  Top 5 features: {sorted(zip(self.feature_names, importance_values), key=lambda x: x[1], reverse=True)[:5]}")

            # Check if all importance values are zero
            if total_importance == 0.0:
                logger.warning("All RandomForest feature importances are zero. Computing fallback importance using correlation analysis.")
                importance_dict = self._compute_fallback_feature_importance(X_train, y_train)
            else:
                logger.info(f"RandomForest feature importance computed successfully. Using RandomForest values.")

        except Exception as e:
            logger.warning(f"Failed to compute RandomForest feature importance: {e}. Using fallback method.")
            importance_dict = self._compute_fallback_feature_importance(X_train, y_train)

        return ModelMetrics(
            train_ndcg=train_ndcg,
            val_ndcg=val_ndcg,
            feature_importance=importance_dict
        )

    def predict(self, X: np.ndarray,
                explain: bool = True) -> List[PredictionResult]:
        """
        Predict risk scores for deployment features.

        Args:
            X: Feature matrix (n_samples, n_features)
            explain: Whether to include SHAP explanations

        Returns:
            List of prediction results with explanations
        """
        if self.model is None:
            raise ValueError("Model not trained. Call train() first.")

        # Scale features
        X_scaled = self.scaler.transform(X)

        # Get predictions
        risk_scores = self.model.predict(X_scaled)

        # Get feature importance/explanations
        results = []
        for i, score in enumerate(risk_scores):
            feature_importance = {}

            if explain and SHAP_AVAILABLE and self.shap_explainer is not None:
                # SHAP explanations
                shap_values = self.shap_explainer.shap_values(X_scaled[i:i+1])
                if isinstance(shap_values, list):
                    shap_values = shap_values[0]  # For multi-class, take first class

                feature_importance = dict(zip(self.feature_names, shap_values[0]))
            else:
                # Fallback to global feature importance
                if hasattr(self.model, 'feature_importance'):
                    feature_importance = dict(zip(self.feature_names, self.model.feature_importance()))
                elif hasattr(self.model, 'feature_importances_'):
                    feature_importance = dict(zip(self.feature_names, self.model.feature_importances_))

            # Sort by importance and take top features
            top_features = dict(sorted(feature_importance.items(),
                                     key=lambda x: abs(x[1]), reverse=True)[:10])

            results.append(PredictionResult(
                risk_score=float(score),
                feature_importance=top_features,
                model_version=self.model_version or "unknown",
                confidence=self._calculate_prediction_confidence(X_scaled[i:i+1], score)
            ))

        return results

    def _compute_fallback_feature_importance(self, X: np.ndarray, y: np.ndarray) -> Dict[str, float]:
        """
        Compute fallback feature importance using correlation with target values.

        Args:
            X: Feature matrix
            y: Target values

        Returns:
            Dictionary of feature names to importance scores
        """
        logger.info(f"Computing fallback feature importance for {len(self.feature_names)} features")

        try:
            import numpy as np
            from scipy.stats import pearsonr

            importance_dict = {}
            correlations_computed = 0
            zero_variance_features = 0

            for i, feature_name in enumerate(self.feature_names):
                feature_values = X[:, i]
                feature_var = np.var(feature_values)

                # Skip features with no variance
                if feature_var == 0:
                    importance_dict[feature_name] = 0.0
                    zero_variance_features += 1
                    logger.debug(f"Feature {feature_name} has zero variance")
                    continue

                # Calculate correlation with target
                try:
                    correlation, p_value = pearsonr(feature_values, y)
                    correlations_computed += 1

                    # Use absolute correlation as importance, weighted by significance
                    if np.isnan(correlation) or np.isnan(p_value):
                        importance = 0.0
                        logger.debug(f"Feature {feature_name}: NaN correlation")
                    else:
                        # Enhanced weighting: correlation * significance * variance factor
                        significance_weight = max(0.1, 1.0 - p_value) if p_value < 1.0 else 0.1
                        variance_factor = min(2.0, np.sqrt(feature_var))  # Boost features with higher variance
                        importance = abs(correlation) * significance_weight * variance_factor

                    importance_dict[feature_name] = importance
                    logger.debug(f"Feature {feature_name}: corr={correlation:.4f}, p={p_value:.4f}, importance={importance:.6f}")

                except Exception as e:
                    logger.debug(f"Failed to compute correlation for feature {feature_name}: {e}")
                    importance_dict[feature_name] = 0.0

            logger.info(f"Fallback computation: {correlations_computed} correlations computed, "
                       f"{zero_variance_features} zero-variance features")

            # Normalize importance scores to sum to 1.0
            total_importance = sum(importance_dict.values())
            logger.info(f"Total raw correlation importance before normalization: {total_importance:.10f}")

            # Enhanced diagnostic for correlation analysis
            valid_correlations = [v for v in importance_dict.values() if v > 0]
            logger.info(f"Correlation analysis results: {len(valid_correlations)} features with positive correlation importance")

            if total_importance > 1e-10:  # More lenient threshold
                importance_dict = {name: score / total_importance for name, score in importance_dict.items()}
                top_features = sorted(importance_dict.items(), key=lambda x: x[1], reverse=True)[:5]
                logger.info(f"SUCCESS: Fallback feature importance computed using correlation. Top features: {top_features}")
            else:
                # If all correlations are zero, use alternative methods
                logger.warning(f"Correlation method failed: all correlations effectively zero (total={total_importance:.10f}). Trying variance-based importance.")

                # Enhanced variance analysis
                variance_importance = {}
                total_variance = 0.0
                variance_stats = []

                for i, feature_name in enumerate(self.feature_names):
                    feature_var = np.var(X[:, i])
                    variance_importance[feature_name] = feature_var
                    total_variance += feature_var
                    variance_stats.append((feature_name, feature_var))

                # Sort by variance for logging
                variance_stats.sort(key=lambda x: x[1], reverse=True)
                logger.info(f"Feature variance analysis - total variance: {total_variance:.10f}")
                logger.info(f"Top 5 variance features: {variance_stats[:5]}")
                logger.info(f"Bottom 5 variance features: {variance_stats[-5:]}")

                if total_variance > 1e-12:  # Even more lenient threshold for variance
                    importance_dict = {name: var / total_variance for name, var in variance_importance.items()}
                    top_features = sorted(importance_dict.items(), key=lambda x: x[1], reverse=True)[:5]
                    logger.info(f"SUCCESS: Using feature variance as importance. Top variance features: {top_features}")
                else:
                    # Last resort: equal importance
                    equal_importance = 1.0 / len(self.feature_names)
                    importance_dict = {name: equal_importance for name in self.feature_names}
                    logger.error(f"FALLBACK TO EQUAL IMPORTANCE: All features have effectively zero variance (total={total_variance:.12f}). "
                               f"Assigning equal importance: {equal_importance:.10f} to each of {len(self.feature_names)} features.")
                    logger.error(f"This suggests either: 1) All feature values are identical, 2) Data preprocessing issue, or 3) Synthetic scoring not working properly.")

            return importance_dict

        except Exception as e:
            logger.error(f"Fallback feature importance calculation failed: {e}")
            import traceback
            logger.error(f"Traceback: {traceback.format_exc()}")
            # Last resort: equal importance
            equal_importance = 1.0 / len(self.feature_names)
            logger.warning(f"Using last resort equal importance: {equal_importance:.6f}")
            return {name: equal_importance for name in self.feature_names}

    def _calculate_prediction_confidence(self, X: np.ndarray, prediction: float) -> float:
        """Calculate prediction confidence score."""
        # Simple confidence based on distance from training data mean
        # In practice, could use more sophisticated uncertainty quantification
        try:
            if hasattr(self.model, 'predict_proba'):
                # For classifiers
                proba = self.model.predict_proba(X)
                return float(np.max(proba))
            else:
                # For regressors, use a simple heuristic
                return min(1.0, max(0.1, 1.0 / (1.0 + abs(prediction - 2.0))))
        except:
            return 0.5  # Default confidence


    def _calculate_regression_ndcg(self, y_true: np.ndarray, y_pred: np.ndarray) -> float:
        """Calculate NDCG for regression (treat as single group)."""
        try:
            if len(np.unique(y_true)) > 1:
                return ndcg_score([y_true], [y_pred])
            else:
                return 0.0
        except:
            return 0.0


    def _initialize_shap_explainer(self, X_sample: np.ndarray) -> None:
        """Initialize SHAP explainer for the trained model."""
        if not SHAP_AVAILABLE:
            return

        try:
            # Use KernelExplainer for sklearn models
            background = shap.sample(X_sample, min(100, len(X_sample)))
            self.shap_explainer = shap.KernelExplainer(self.model.predict, background)
            logger.info("SHAP explainer initialized successfully")

        except Exception as e:
            logger.warning(f"Failed to initialize SHAP explainer: {e}")
            self.shap_explainer = None

    def get_global_feature_importance(self) -> Dict[str, float]:
        """Get global feature importance from the trained model."""
        if self.training_metrics and self.training_metrics.feature_importance:
            return self.training_metrics.feature_importance
        return {}

    def save_model(self, filepath: str) -> None:
        """Save the trained model to file."""
        if self.model is None:
            raise ValueError("No model to save. Train the model first.")

        model_data = {
            'model': self.model,
            'scaler': self.scaler,
            'feature_names': self.feature_names,
            'model_version': self.model_version,
            'algorithm': self.algorithm,
            'config': self.config,
            'training_metrics': self.training_metrics
        }

        joblib.dump(model_data, filepath)
        logger.info(f"Model saved to {filepath}")

    def load_model(self, filepath: str) -> None:
        """Load a trained model from file."""
        model_data = joblib.load(filepath)

        self.model = model_data['model']
        self.scaler = model_data['scaler']
        self.feature_names = model_data['feature_names']
        self.model_version = model_data['model_version']
        self.algorithm = model_data['algorithm']
        self.config = model_data.get('config', self._default_config())
        self.training_metrics = model_data.get('training_metrics')

        # Reinitialize SHAP explainer if available
        if SHAP_AVAILABLE and self.scaler is not None:
            try:
                # Create sample data for explainer initialization
                sample_data = np.random.randn(10, len(self.feature_names))
                sample_scaled = self.scaler.transform(sample_data)
                self._initialize_shap_explainer(sample_scaled)
            except Exception as e:
                logger.warning(f"Failed to reinitialize SHAP explainer: {e}")

        logger.info(f"Model loaded from {filepath}")

    def get_model_info(self) -> Dict[str, Any]:
        """Get information about the current model."""
        def convert_numpy_types(obj):
            """Convert numpy types to Python native types for JSON serialization."""
            if hasattr(obj, 'item'):  # numpy scalar
                return obj.item()
            elif isinstance(obj, dict):
                return {k: convert_numpy_types(v) for k, v in obj.items()}
            elif isinstance(obj, list):
                return [convert_numpy_types(v) for v in obj]
            else:
                return obj

        # Convert training_metrics to dict and handle numpy types
        training_metrics_dict = None
        if self.training_metrics:
            training_metrics_dict = asdict(self.training_metrics)
            training_metrics_dict = convert_numpy_types(training_metrics_dict)

        # Add feature importance diagnostics
        feature_importance_info = {}
        if self.training_metrics and self.training_metrics.feature_importance:
            importance_values = list(self.training_metrics.feature_importance.values())
            feature_importance_info = {
                'total_importance': sum(importance_values),
                'max_importance': max(importance_values) if importance_values else 0.0,
                'min_importance': min(importance_values) if importance_values else 0.0,
                'non_zero_features': sum(1 for v in importance_values if v > 0),
                'zero_features': sum(1 for v in importance_values if v == 0),
                'top_5_features': sorted(self.training_metrics.feature_importance.items(),
                                       key=lambda x: x[1], reverse=True)[:5]
            }

        return {
            'model_version': self.model_version,
            'algorithm': self.algorithm,
            'feature_count': len(self.feature_names) if self.feature_names else 0,
            'feature_names': self.feature_names,
            'training_metrics': training_metrics_dict,
            'feature_importance_diagnostics': feature_importance_info,
            'shap_available': self.shap_explainer is not None,
            'trained': self.model is not None
        }

    def save_model_to_storage(self, storage_manager, model_id: str, description: str = "", tags: Dict[str, str] = None) -> bool:
        """Save model using storage manager with metadata tracking."""
        if self.model is None:
            raise ValueError("No model to save. Train the model first.")

        from src.storage.model_storage import ModelMetadata

        # Serialize model data to bytes
        model_data = {
            'model': self.model,
            'scaler': self.scaler,
            'feature_names': self.feature_names,
            'model_version': self.model_version,
            'algorithm': self.algorithm,
            'config': self.config,
            'training_metrics': self.training_metrics
        }

        # Convert to bytes using joblib
        buffer = io.BytesIO()
        joblib.dump(model_data, buffer)
        model_bytes = buffer.getvalue()

        # Create metadata
        version = self.model_version.split('_')[-1] if '_' in self.model_version else "1"
        performance_metrics = {}
        if self.training_metrics:
            performance_metrics = {
                'train_ndcg': getattr(self.training_metrics, 'train_ndcg', 0.0),
                'val_ndcg': getattr(self.training_metrics, 'val_ndcg', 0.0)
            }

        metadata = ModelMetadata(
            model_id=model_id,
            version=version,
            algorithm=self.algorithm,
            feature_count=len(self.feature_names) if self.feature_names else 0,
            training_timestamp=datetime.now(timezone.utc).isoformat(),
            model_size_bytes=len(model_bytes),
            checksum="",  # Will be calculated by storage manager
            performance_metrics=performance_metrics,
            config=self.config,
            tags=tags or {},
            description=description
        )

        success = storage_manager.save_model(model_bytes, metadata)
        if success:
            logger.info(f"Model {model_id} v{version} saved to storage successfully")
        else:
            logger.error(f"Failed to save model {model_id} to storage")

        return success

    def load_model_from_storage(self, storage_manager, model_id: str, version: Optional[str] = None) -> bool:
        """Load model from storage manager."""
        try:
            model_bytes, metadata = storage_manager.load_model(model_id, version)

            # Deserialize model data
            buffer = io.BytesIO(model_bytes)
            model_data = joblib.load(buffer)

            # Restore model state
            self.model = model_data['model']
            self.scaler = model_data['scaler']
            self.feature_names = model_data['feature_names']
            self.model_version = model_data['model_version']
            self.algorithm = model_data['algorithm']
            self.config = model_data.get('config', self._default_config())
            self.training_metrics = model_data.get('training_metrics')

            # Reinitialize SHAP explainer if available
            if SHAP_AVAILABLE and self.scaler is not None:
                try:
                    # Create sample data for explainer initialization
                    sample_data = np.random.randn(10, len(self.feature_names))
                    sample_scaled = self.scaler.transform(sample_data)
                    self._initialize_shap_explainer(sample_scaled)
                except Exception as e:
                    logger.warning(f"Failed to reinitialize SHAP explainer: {e}")

            logger.info(f"Model {model_id} v{metadata.version} loaded from storage successfully")
            return True

        except Exception as e:
            logger.error(f"Failed to load model {model_id} from storage: {e}")
            return False

    @classmethod
    def create_from_storage(cls, storage_manager, model_id: str, version: Optional[str] = None, config: Optional[Dict[str, Any]] = None):
        """Create a new RiskRankingModel instance by loading from storage."""
        model = cls(config)
        if model.load_model_from_storage(storage_manager, model_id, version):
            return model
        else:
            raise ValueError(f"Failed to load model {model_id} from storage")