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
        self.algorithm = self.config.get('model', {}).get('algorithm', 'lightgbm_ranker')

    def _default_config(self) -> Dict[str, Any]:
        """Default model configuration."""
        return {
            'model': {
                'algorithm': 'lightgbm_ranker',
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
        if training_config.get('early_stopping_rounds'):
            callbacks.append(lgb.early_stopping(training_config['early_stopping_rounds'], verbose=False))

        self.model = lgb.train(
            params=params,
            train_set=train_data,
            valid_sets=[train_data, val_data],
            valid_names=['train', 'valid'],
            num_boost_round=training_config.get('max_iterations', 100),
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

        # Feature importance
        importance_dict = dict(zip(self.feature_names, self.model.feature_importances_))

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
        n_val_groups = int(n_groups * val_split)

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
        return {
            'model_version': self.model_version,
            'algorithm': self.algorithm,
            'feature_count': len(self.feature_names) if self.feature_names else 0,
            'feature_names': self.feature_names,
            'training_metrics': asdict(self.training_metrics) if self.training_metrics else None,
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