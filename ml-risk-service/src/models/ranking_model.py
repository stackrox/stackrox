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
import lightgbm as lgb

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
    """Model performance metrics."""
    train_ndcg: float = 0.0
    val_ndcg: float = 0.0
    train_auc: float = 0.0
    val_auc: float = 0.0
    training_loss: float = 0.0
    epochs_completed: int = 0
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
    Supports both LightGBM ranker and traditional regression approaches.
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
                'lightgbm_params': {
                    'objective': 'lambdarank',
                    'metric': 'ndcg',
                    'num_leaves': 31,
                    'learning_rate': 0.1,
                    'feature_fraction': 0.9,
                    'bagging_fraction': 0.8,
                    'bagging_freq': 5,
                    'verbose': 0,
                    'force_row_wise': True
                },
                'explainability': {
                    'shap_enabled': True,
                    'top_features': 10
                }
            },
            'training': {
                'batch_size': 1000,
                'max_iterations': 100,
                'early_stopping_rounds': 10
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
                train_auc=0.0,
                val_auc=0.0,
                training_loss=0.0,
                epochs_completed=0,
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

        # For LightGBM ranking, ensure all scores are unique by adding small epsilon
        if self.algorithm == 'lightgbm_ranker':
            # Add tiny random noise to ensure all values are unique
            epsilon = 1e-8
            np.random.seed(42)  # Deterministic noise
            y = y + np.random.uniform(-epsilon, epsilon, size=len(y))
            logger.info(f"Added epsilon noise for unique ranking values: {len(np.unique(y))} unique values")

        # Split data
        val_split = self.config.get('model', {}).get('validation_split', 0.2)
        random_state = self.config.get('model', {}).get('random_state', 42)

        if groups is not None and self.algorithm == 'lightgbm_ranker':
            # For ranking, split by groups
            X_train, X_val, y_train, y_val, groups_train, groups_val = self._split_ranking_data(
                X, y, groups, val_split, random_state)
        else:
            X_train, X_val, y_train, y_val = train_test_split(
                X, y, test_size=val_split, random_state=random_state)
            groups_train = groups_val = None

        # Scale features
        self.scaler = StandardScaler()
        X_train_scaled = self.scaler.fit_transform(X_train)
        X_val_scaled = self.scaler.transform(X_val)

        # Convert to integer ranks ONLY for LightGBM, sklearn works with float scores
        if self.algorithm == 'lightgbm_ranker':
            # Use simple sequential mapping to ensure contiguous labels
            unique_train_sorted = np.unique(y_train)
            unique_val_sorted = np.unique(y_val)

            # Create explicit mapping to ensure contiguous labels 0, 1, 2, ..., n-1
            train_mapping = {orig_val: i for i, orig_val in enumerate(unique_train_sorted)}
            val_mapping = {orig_val: i for i, orig_val in enumerate(unique_val_sorted)}

            y_train = np.array([train_mapping[val] for val in y_train], dtype=np.int32)
            y_val = np.array([val_mapping[val] for val in y_val], dtype=np.int32)

            logger.info(f"Sequential mapping - Train: {len(unique_train_sorted)} unique -> 0-{len(unique_train_sorted)-1}, Val: {len(unique_val_sorted)} unique -> 0-{len(unique_val_sorted)-1}")
            logger.info(f"Final ranking - Train: {y_train.min()}-{y_train.max()} ({len(np.unique(y_train))} unique), Val: {y_val.min()}-{y_val.max()} ({len(np.unique(y_val))} unique)")
        else:
            # For sklearn, use float scores directly
            logger.info(f"Using float scores - Train: {y_train.min():.6f}-{y_train.max():.6f}, Val: {y_val.min():.6f}-{y_val.max():.6f}")

        # Train model based on algorithm
        if self.algorithm == 'lightgbm_ranker':
            metrics = self._train_lightgbm_ranker(
                X_train_scaled, y_train, X_val_scaled, y_val,
                groups_train, groups_val)
        elif self.algorithm == 'sklearn_ranksvm':
            metrics = self._train_sklearn_ranksvm(
                X_train_scaled, y_train, X_val_scaled, y_val)
        else:
            raise ValueError(f"Unknown algorithm: {self.algorithm}")

        self.training_metrics = metrics
        self.model_version = f"{self.algorithm}_{datetime.now().strftime('%Y%m%d_%H%M%S')}"

        # Initialize SHAP explainer
        if SHAP_AVAILABLE and self.config.get('model', {}).get('explainability', {}).get('shap_enabled', True):
            self._initialize_shap_explainer(X_train_scaled)

        logger.info(f"Training complete. Validation NDCG: {metrics.val_ndcg:.4f}")
        return metrics

    def _train_lightgbm_ranker(self, X_train: np.ndarray, y_train: np.ndarray,
                              X_val: np.ndarray, y_val: np.ndarray,
                              groups_train: Optional[np.ndarray],
                              groups_val: Optional[np.ndarray]) -> ModelMetrics:
        """Train LightGBM ranker model."""

        params = self.config.get('model', {}).get('lightgbm_params', {})
        training_config = self.config.get('training', {})

        # Create datasets
        train_data = lgb.Dataset(X_train, label=y_train, group=groups_train,
                                feature_name=self.feature_names)
        val_data = lgb.Dataset(X_val, label=y_val, group=groups_val,
                              feature_name=self.feature_names)

        # Train model
        callbacks = []

        # Use conservative early stopping to ensure minimum learning
        early_stopping_rounds = training_config.get('early_stopping_rounds', 10)
        min_iterations = max(50, early_stopping_rounds * 2)  # Ensure at least 50 iterations for feature learning

        if early_stopping_rounds and len(y_train) > 100:  # Only use early stopping with sufficient data
            callbacks.append(lgb.early_stopping(early_stopping_rounds, verbose=False))
        else:
            logger.info("Skipping early stopping due to insufficient data - training with fixed iterations")

        # Ensure minimum iterations for meaningful training and feature importance learning
        max_iterations = max(min_iterations, training_config.get('max_iterations', 100))

        self.model = lgb.train(
            params=params,
            train_set=train_data,
            valid_sets=[train_data, val_data],
            valid_names=['train', 'valid'],
            num_boost_round=max_iterations,
            callbacks=callbacks
        )

        # Calculate metrics
        train_pred = self.model.predict(X_train)
        val_pred = self.model.predict(X_val)

        # NDCG scores
        train_ndcg = self._calculate_ndcg(y_train, train_pred, groups_train)
        val_ndcg = self._calculate_ndcg(y_val, val_pred, groups_val)

        # Feature importance
        importance_dict = dict(zip(self.feature_names, self.model.feature_importance()))

        return ModelMetrics(
            train_ndcg=train_ndcg,
            val_ndcg=val_ndcg,
            epochs_completed=self.model.num_trees(),
            feature_importance=importance_dict
        )

    def _train_sklearn_ranksvm(self, X_train: np.ndarray, y_train: np.ndarray,
                              X_val: np.ndarray, y_val: np.ndarray) -> ModelMetrics:
        """Train sklearn-based ranking model (fallback)."""

        from sklearn.svm import SVR
        from sklearn.ensemble import RandomForestRegressor

        # Use RandomForest as a ranking approximation
        self.model = RandomForestRegressor(
            n_estimators=100,
            random_state=self.config.get('model', {}).get('random_state', 42),
            n_jobs=-1
        )

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

            # Check if all importance values are zero
            total_importance = sum(importance_values)
            if total_importance == 0.0:
                logger.warning("All feature importances are zero. Computing fallback importance using feature variance.")
                importance_dict = self._compute_fallback_feature_importance(X_train, y_train)
            else:
                logger.info(f"Feature importance computed successfully. Total importance: {total_importance:.6f}")

        except Exception as e:
            logger.warning(f"Failed to compute feature importance: {e}. Using fallback method.")
            importance_dict = self._compute_fallback_feature_importance(X_train, y_train)

        return ModelMetrics(
            train_ndcg=train_ndcg,
            val_ndcg=val_ndcg,
            epochs_completed=100,
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
        try:
            import numpy as np
            from scipy.stats import pearsonr

            importance_dict = {}

            for i, feature_name in enumerate(self.feature_names):
                feature_values = X[:, i]

                # Skip features with no variance
                if np.var(feature_values) == 0:
                    importance_dict[feature_name] = 0.0
                    continue

                # Calculate correlation with target
                try:
                    correlation, p_value = pearsonr(feature_values, y)
                    # Use absolute correlation as importance, weighted by significance
                    if np.isnan(correlation) or np.isnan(p_value):
                        importance = 0.0
                    else:
                        # Weight by inverse p-value (more significant = higher importance)
                        significance_weight = max(0.1, 1.0 - p_value) if p_value < 1.0 else 0.1
                        importance = abs(correlation) * significance_weight

                    importance_dict[feature_name] = importance

                except Exception as e:
                    logger.debug(f"Failed to compute correlation for feature {feature_name}: {e}")
                    importance_dict[feature_name] = 0.0

            # Normalize importance scores to sum to 1.0
            total_importance = sum(importance_dict.values())
            if total_importance > 0:
                importance_dict = {name: score / total_importance for name, score in importance_dict.items()}
                logger.info(f"Fallback feature importance computed using correlation. Top features: "
                           f"{sorted(importance_dict.items(), key=lambda x: x[1], reverse=True)[:5]}")
            else:
                # If all correlations are zero, assign equal importance
                equal_importance = 1.0 / len(self.feature_names)
                importance_dict = {name: equal_importance for name in self.feature_names}
                logger.warning("All correlations are zero. Assigning equal importance to all features.")

            return importance_dict

        except Exception as e:
            logger.error(f"Fallback feature importance calculation failed: {e}")
            # Last resort: equal importance
            equal_importance = 1.0 / len(self.feature_names)
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

    def _calculate_ndcg(self, y_true: np.ndarray, y_pred: np.ndarray,
                       groups: Optional[np.ndarray]) -> float:
        """Calculate NDCG score for ranking evaluation."""
        if groups is None:
            return self._calculate_regression_ndcg(y_true, y_pred)

        # Calculate NDCG per group and average
        ndcg_scores = []
        start_idx = 0

        for group_size in groups:
            end_idx = start_idx + group_size
            group_true = y_true[start_idx:end_idx]
            group_pred = y_pred[start_idx:end_idx]

            if len(np.unique(group_true)) > 1:  # Only if there's variance in scores
                try:
                    # Reshape for ndcg_score function
                    ndcg = ndcg_score([group_true], [group_pred])
                    ndcg_scores.append(ndcg)
                except:
                    pass  # Skip groups with issues

            start_idx = end_idx

        return np.mean(ndcg_scores) if ndcg_scores else 0.0

    def _calculate_regression_ndcg(self, y_true: np.ndarray, y_pred: np.ndarray) -> float:
        """Calculate NDCG for regression (treat as single group)."""
        try:
            if len(np.unique(y_true)) > 1:
                return ndcg_score([y_true], [y_pred])
            else:
                return 0.0
        except:
            return 0.0

    def _split_ranking_data(self, X: np.ndarray, y: np.ndarray, groups: np.ndarray,
                           val_split: float, random_state: int) -> Tuple[np.ndarray, ...]:
        """Split ranking data maintaining group structure."""

        # Split groups, not individual samples
        n_groups = len(groups)

        # Handle single group case - fall back to sample-based split
        if n_groups == 1:
            logger.info(f"Single group detected ({groups[0]} samples), using sample-based validation split")
            X_train, X_val, y_train, y_val = train_test_split(
                X, y, test_size=val_split, random_state=random_state)

            # Create group arrays for the splits
            groups_train = np.array([len(y_train)])
            groups_val = np.array([len(y_val)])

            return X_train, X_val, y_train, y_val, groups_train, groups_val

        # Regular group-based split for multiple groups
        # Ensure at least 1 group goes to validation if we have multiple groups
        n_val_groups = max(1, int(n_groups * val_split))

        np.random.seed(random_state)
        val_group_indices = np.random.choice(n_groups, n_val_groups, replace=False)
        train_group_indices = np.setdiff1d(np.arange(n_groups), val_group_indices)

        # Map group indices to sample indices
        train_indices = []
        val_indices = []
        start_idx = 0

        for i, group_size in enumerate(groups):
            end_idx = start_idx + group_size
            if i in train_group_indices:
                train_indices.extend(range(start_idx, end_idx))
            else:
                val_indices.extend(range(start_idx, end_idx))
            start_idx = end_idx

        # Create splits
        X_train, X_val = X[train_indices], X[val_indices]
        y_train, y_val = y[train_indices], y[val_indices]
        groups_train = groups[train_group_indices]
        groups_val = groups[val_group_indices]

        logger.info(f"Group-based split: {len(groups_train)} training groups ({len(y_train)} samples), "
                   f"{len(groups_val)} validation groups ({len(y_val)} samples)")

        return X_train, X_val, y_train, y_val, groups_train, groups_val

    def _initialize_shap_explainer(self, X_sample: np.ndarray) -> None:
        """Initialize SHAP explainer for the trained model."""
        if not SHAP_AVAILABLE:
            return

        try:
            if hasattr(self.model, 'predict'):
                # For LightGBM and sklearn models
                if isinstance(self.model, lgb.Booster):
                    self.shap_explainer = shap.TreeExplainer(self.model)
                else:
                    # Use KernelExplainer for other models
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
                'val_ndcg': getattr(self.training_metrics, 'val_ndcg', 0.0),
                'train_auc': getattr(self.training_metrics, 'train_auc', 0.0),
                'val_auc': getattr(self.training_metrics, 'val_auc', 0.0),
                'training_loss': getattr(self.training_metrics, 'training_loss', 0.0),
                'epochs_completed': getattr(self.training_metrics, 'epochs_completed', 0)
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