"""
Complete training pipeline for ML risk ranking system.
"""

import logging
import json
import os
import yaml
import hashlib
import io
import joblib
from typing import Dict, Any, List, Optional
import numpy as np
from datetime import datetime
import pickle

from .data_loader import TrainingDataLoader, JSONTrainingDataGenerator
from .baseline_reproducer import BaselineReproducer
from src.models.ranking_model import RiskRankingModel
from src.models.feature_importance import FeatureImportanceAnalyzer
from src.storage.model_storage import ModelMetadata

logger = logging.getLogger(__name__)


class TrainingPipeline:
    """
    Complete training pipeline for ML risk ranking system.
    Handles data loading, model training, evaluation, and model persistence.
    """

    def __init__(self, config_path: Optional[str] = None, storage_manager=None):
        self.config = self._load_config(config_path)
        self.data_loader = TrainingDataLoader(self.config)
        self.baseline_reproducer = BaselineReproducer()
        self.model = RiskRankingModel(self.config)
        self.feature_analyzer = FeatureImportanceAnalyzer()
        self.storage_manager = storage_manager

        # Training state
        self.training_data = None
        self.model_trained = False

    def _load_config(self, config_path: Optional[str]) -> Dict[str, Any]:
        """Load training configuration."""
        if config_path and os.path.exists(config_path):
            with open(config_path, 'r') as f:
                return yaml.safe_load(f)
        else:
            # Default configuration
            return {
                'data': {
                    'train_file': None,
                    'validation_split': 0.2,
                    'max_examples': None
                },
                'model': {
                    'algorithm': 'lightgbm_ranker',
                    'validation_split': 0.2,
                    'random_state': 42
                },
                'training': {
                    'batch_size': 1000,
                    'max_iterations': 100,
                    'early_stopping_rounds': 10
                },
                'output': {
                    'model_dir': './models',
                    'reports_dir': './reports'
                }
            }

    def run_full_pipeline(self, training_data_file: str) -> Dict[str, Any]:
        """
        Run the complete training pipeline.

        Args:
            training_data_file: Path to training data JSON file

        Returns:
            Pipeline results and metrics
        """
        logger.info("Starting full training pipeline")
        pipeline_results = {
            'success': False,
            'pipeline_start_time': datetime.now().isoformat(),
            'stages_completed': []
        }

        try:
            # Stage 1: Data loading and validation
            logger.info("Stage 1: Loading and validating training data")
            data_results = self._load_and_validate_data(training_data_file)
            pipeline_results['data_validation'] = data_results
            pipeline_results['stages_completed'].append('data_loading')

            if not data_results['valid']:
                raise ValueError(f"Data validation failed: {data_results.get('error', 'Unknown error')}")

            # Stage 2: Baseline reproduction validation
            logger.info("Stage 2: Validating baseline reproduction")
            baseline_results = self._validate_baseline_reproduction()
            pipeline_results['baseline_validation'] = baseline_results
            pipeline_results['stages_completed'].append('baseline_validation')

            # Stage 3: Model training
            logger.info("Stage 3: Training ML model")
            training_results = self._train_model()
            pipeline_results['model_training'] = training_results
            pipeline_results['stages_completed'].append('model_training')

            # Stage 4: Model evaluation
            logger.info("Stage 4: Evaluating model performance")
            evaluation_results = self._evaluate_model()
            pipeline_results['model_evaluation'] = evaluation_results
            pipeline_results['stages_completed'].append('model_evaluation')

            # Stage 5: Feature importance analysis
            logger.info("Stage 5: Analyzing feature importance")
            importance_results = self._analyze_feature_importance()
            pipeline_results['feature_analysis'] = importance_results
            pipeline_results['stages_completed'].append('feature_analysis')

            # Stage 6: Model saving and reporting
            logger.info("Stage 6: Saving model and generating reports")
            save_results = self._save_model_and_reports()
            pipeline_results['model_saving'] = save_results
            pipeline_results['stages_completed'].append('model_saving')

            pipeline_results['success'] = True
            pipeline_results['pipeline_end_time'] = datetime.now().isoformat()

            logger.info("Training pipeline completed successfully")

        except Exception as e:
            logger.error(f"Training pipeline failed: {e}")
            pipeline_results['error'] = str(e)
            pipeline_results['pipeline_end_time'] = datetime.now().isoformat()

        return pipeline_results

    def _load_and_validate_data(self, data_file: str) -> Dict[str, Any]:
        """Load and validate training data."""
        try:
            # Load training data
            self.training_data = self.data_loader.load_from_json(data_file)

            # Limit examples if configured
            max_examples = self.config.get('data', {}).get('max_examples')
            if max_examples and len(self.training_data) > max_examples:
                logger.info(f"Limiting training data to {max_examples} examples")
                self.training_data = self.training_data[:max_examples]

            # Validate data quality
            validation_report = self.data_loader.validate_training_data(self.training_data)

            return {
                'valid': validation_report['valid'],
                'total_examples': len(self.training_data),
                'validation_report': validation_report
            }

        except Exception as e:
            return {'valid': False, 'error': str(e)}

    def _validate_baseline_reproduction(self) -> Dict[str, Any]:
        """Validate that we can reproduce baseline risk scores."""
        try:
            # Use a subset for baseline validation
            validation_subset = self.training_data[:min(100, len(self.training_data))]

            # Convert training examples back to baseline format
            baseline_test_data = []
            for example in validation_subset:
                if 'deployment_id' in example:
                    # Mock deployment record for baseline validation
                    record = {
                        'deployment': {'id': example['deployment_id']},
                        'images': [],
                        'alerts': [],
                        'current_risk_score': example['risk_score']
                    }
                    baseline_test_data.append(record)

            if not baseline_test_data:
                return {'valid': False, 'error': 'No data suitable for baseline validation'}

            # Run baseline reproduction validation
            validation_results = self.baseline_reproducer.validate_baseline_reproduction(baseline_test_data)

            return {
                'valid': validation_results.get('assessment', 'POOR') in ['EXCELLENT', 'GOOD', 'ACCEPTABLE'],
                'validation_results': validation_results
            }

        except Exception as e:
            logger.warning(f"Baseline validation failed: {e}")
            return {'valid': True, 'warning': str(e)}  # Don't fail pipeline

    def _train_model(self) -> Dict[str, Any]:
        """Train the ML ranking model."""
        try:
            # Create ranking dataset
            X, y, groups = self.data_loader.create_ranking_dataset(self.training_data)

            # Get feature names
            if self.training_data:
                feature_names = sorted(self.training_data[0]['features'].keys())
            else:
                raise ValueError("No training data available")

            # Train model
            training_metrics = self.model.train(X, y, groups, feature_names)

            self.model_trained = True

            return {
                'success': True,
                'training_metrics': {
                    'train_ndcg': training_metrics.train_ndcg,
                    'val_ndcg': training_metrics.val_ndcg,
                    'epochs_completed': training_metrics.epochs_completed
                },
                'dataset_info': {
                    'total_examples': len(self.training_data),
                    'feature_count': X.shape[1],
                    'groups_count': len(groups) if groups is not None else 0
                }
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def _evaluate_model(self) -> Dict[str, Any]:
        """Evaluate trained model performance."""
        try:
            if not self.model_trained:
                raise ValueError("Model not trained")

            # Create test dataset (using same data for now - in practice would use holdout set)
            X, y, groups = self.data_loader.create_ranking_dataset(self.training_data)

            # Get predictions
            predictions = self.model.predict(X, explain=False)
            predicted_scores = [pred.risk_score for pred in predictions]

            # Calculate evaluation metrics
            from sklearn.metrics import mean_squared_error, mean_absolute_error
            import numpy as np

            mse = mean_squared_error(y, predicted_scores)
            mae = mean_absolute_error(y, predicted_scores)
            correlation = np.corrcoef(y, predicted_scores)[0, 1]

            # NDCG calculation
            ndcg = self.model._calculate_ndcg(y, np.array(predicted_scores), groups)

            return {
                'success': True,
                'metrics': {
                    'mse': float(mse),
                    'mae': float(mae),
                    'rmse': float(np.sqrt(mse)),
                    'correlation': float(correlation),
                    'ndcg': float(ndcg)
                },
                'prediction_stats': {
                    'min_pred': float(np.min(predicted_scores)),
                    'max_pred': float(np.max(predicted_scores)),
                    'mean_pred': float(np.mean(predicted_scores)),
                    'std_pred': float(np.std(predicted_scores))
                }
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def _analyze_feature_importance(self) -> Dict[str, Any]:
        """Analyze feature importance and generate explanations."""
        try:
            if not self.model_trained:
                raise ValueError("Model not trained")

            # Global feature importance analysis
            feature_names = self.model.feature_names
            global_analysis = self.feature_analyzer.analyze_global_importance(
                self.model.model, feature_names)

            # Generate sample explanations
            X, y, _ = self.data_loader.create_ranking_dataset(self.training_data[:10])  # Sample of 10
            sample_explanations = []

            for i in range(min(10, len(X))):
                prediction = self.model.predict(X[i:i+1], explain=True)[0]
                explanation = self.feature_analyzer.explain_prediction(
                    self.model.model, X[i], prediction.risk_score,
                    feature_names, f"sample-{i}", self.model.shap_explainer)
                sample_explanations.append(explanation)

            return {
                'success': True,
                'global_analysis': global_analysis,
                'sample_explanations_count': len(sample_explanations),
                'top_global_features': global_analysis.get('top_features', [])[:5],
                'top_categories': list(global_analysis.get('category_importance', {}).keys())[:5]
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def _save_model_and_reports(self) -> Dict[str, Any]:
        """Save trained model using structured storage and generate reports."""
        try:
            timestamp = datetime.now().strftime('%Y%m%d_%H%M%S')

            # Save model using storage manager if available
            if self.storage_manager:
                # Create model metadata
                model_info = self.model.get_model_info()
                model_id = "stackrox-risk-model"
                version = timestamp

                # Create ModelMetadata - sanitize performance metrics for JSON serialization
                performance_metrics = model_info.get('training_metrics', {})
                sanitized_metrics = self._sanitize_float_values(performance_metrics)

                # Create structured model data matching the expected format
                model_data = {
                    'model': self.model.model,
                    'scaler': self.model.scaler,
                    'feature_names': self.model.feature_names,
                    'model_version': self.model.model_version,
                    'algorithm': self.model.algorithm,
                    'config': self.model.config,
                    'training_metrics': self.model.training_metrics
                }

                # Serialize the structured data using joblib
                model_buffer = io.BytesIO()
                joblib.dump(model_data, model_buffer)
                model_data_bytes = model_buffer.getvalue()

                metadata = ModelMetadata(
                    model_id=model_id,
                    version=version,
                    algorithm=model_info.get('algorithm', 'lightgbm_ranker'),
                    feature_count=int(model_info.get('feature_count', 0)),
                    training_timestamp=datetime.now().isoformat(),
                    model_size_bytes=len(model_data_bytes),
                    checksum=hashlib.md5(model_data_bytes).hexdigest(),
                    performance_metrics=sanitized_metrics,
                    config=self.config,
                    created_by="training-pipeline",
                    semantic_version=f"1.0.{len(str(timestamp))}",
                    status="production",
                    deployment_stage="development"
                )

                # Save using storage manager
                success = self.storage_manager.save_model(model_data_bytes, metadata)

                if success:
                    model_file = f"models/{model_id}/v{version}/model.joblib"
                    logger.info(f"Model saved using storage manager: {model_id} v{version}")
                else:
                    raise Exception("Failed to save model using storage manager")
            else:
                # Fallback to old method if no storage manager
                model_dir = self.config.get('output', {}).get('model_dir', './models')
                os.makedirs(model_dir, exist_ok=True)
                model_file = os.path.join(model_dir, f'risk_ranking_model_{timestamp}.pkl')
                self.model.save_model(model_file)

            # Generate training report (keep existing behavior)
            reports_dir = self.config.get('output', {}).get('reports_dir', './reports')
            os.makedirs(reports_dir, exist_ok=True)

            training_report = {
                'model_info': self.model.get_model_info(),
                'training_config': self.config,
                'data_stats': {
                    'total_examples': len(self.training_data) if self.training_data else 0
                },
                'timestamp': datetime.now().isoformat()
            }

            report_file = os.path.join(reports_dir, f'training_report_{timestamp}.json')
            with open(report_file, 'w') as f:
                json.dump(training_report, f, indent=2, default=str)

            return {
                'success': True,
                'model_file': model_file,
                'report_file': report_file,
                'model_version': self.model.model_version,
                'model_id': model_id if self.storage_manager else None,
                'version': version if self.storage_manager else None
            }

        except Exception as e:
            logger.error(f"Failed to save model and reports: {e}")
            return {'success': False, 'error': str(e)}

    def create_sample_training_data(self, output_file: str, num_examples: int = 1000) -> Dict[str, Any]:
        """
        Create sample training data for testing the pipeline.

        Args:
            output_file: Path to output JSON file
            num_examples: Number of examples to generate

        Returns:
            Generation results
        """
        try:
            generator = JSONTrainingDataGenerator()
            generator.generate_sample_training_data(output_file, num_examples)

            # Validate generated data
            test_data = self.data_loader.load_from_json(output_file)
            validation_report = self.data_loader.validate_training_data(test_data)

            return {
                'success': True,
                'output_file': output_file,
                'examples_generated': num_examples,
                'validation_passed': validation_report['valid']
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def quick_test_pipeline(self) -> Dict[str, Any]:
        """
        Run a quick test of the training pipeline with sample data.

        Returns:
            Test results
        """
        logger.info("Running quick pipeline test")

        try:
            # Generate small sample dataset
            sample_file = "/tmp/quick_test_training_data.json"
            generation_results = self.create_sample_training_data(sample_file, 50)

            if not generation_results['success']:
                return {'success': False, 'error': f"Sample data generation failed: {generation_results['error']}"}

            # Run pipeline
            pipeline_results = self.run_full_pipeline(sample_file)

            # Cleanup
            if os.path.exists(sample_file):
                os.remove(sample_file)

            return {
                'success': pipeline_results['success'],
                'pipeline_results': pipeline_results,
                'test_completed': True
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def load_trained_model(self, model_file: str) -> Dict[str, Any]:
        """
        Load a previously trained model.

        Args:
            model_file: Path to saved model file

        Returns:
            Load results
        """
        try:
            self.model.load_model(model_file)
            self.model_trained = True

            return {
                'success': True,
                'model_info': self.model.get_model_info(),
                'loaded_from': model_file
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def predict_deployment_risk(self, features: Dict[str, float],
                              explain: bool = True) -> Dict[str, Any]:
        """
        Predict risk for a single deployment.

        Args:
            features: Feature dictionary
            explain: Whether to include explanations

        Returns:
            Prediction results
        """
        try:
            if not self.model_trained:
                raise ValueError("No trained model available")

            # Convert features to array
            feature_names = self.model.feature_names
            feature_vector = np.array([features.get(name, 0.0) for name in feature_names])

            # Get prediction
            predictions = self.model.predict(feature_vector.reshape(1, -1), explain=explain)
            prediction = predictions[0]

            # Generate explanation if requested
            explanation = None
            if explain:
                explanation = self.feature_analyzer.explain_prediction(
                    self.model.model, feature_vector, prediction.risk_score,
                    feature_names, "single_prediction", self.model.shap_explainer)

            return {
                'success': True,
                'risk_score': prediction.risk_score,
                'model_version': prediction.model_version,
                'confidence': prediction.confidence,
                'feature_importance': prediction.feature_importance,
                'explanation': explanation.explanation_summary if explanation else None
            }

        except Exception as e:
            return {'success': False, 'error': str(e)}

    def _sanitize_float_values(self, data: Any) -> Any:
        """
        Recursively sanitize values to ensure JSON serialization compatibility.
        Converts NaN, infinity, numpy types to Python native types.

        Args:
            data: Data structure to sanitize

        Returns:
            Sanitized data structure
        """
        import math
        import numpy as np

        if isinstance(data, dict):
            return {key: self._sanitize_float_values(value) for key, value in data.items()}
        elif isinstance(data, list):
            return [self._sanitize_float_values(item) for item in data]
        elif isinstance(data, float):
            if math.isnan(data):
                return None
            elif math.isinf(data):
                return "Infinity" if data > 0 else "-Infinity"
            else:
                return data
        elif hasattr(data, 'dtype') and hasattr(data, 'item'):  # numpy scalar
            if np.issubdtype(data.dtype, np.integer):
                return int(data.item())
            elif np.issubdtype(data.dtype, np.floating):
                val = float(data.item())
                if math.isnan(val):
                    return None
                elif math.isinf(val):
                    return "Infinity" if val > 0 else "-Infinity"
                else:
                    return val
            else:
                return data.item()
        elif isinstance(data, (list, tuple)) and hasattr(data, '__iter__'):
            return [self._sanitize_float_values(item) for item in data]
        else:
            return data