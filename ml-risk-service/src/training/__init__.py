"""
ML training pipeline for risk ranking models.

This module contains offline training pipeline components that are used
to train and evaluate risk ranking models from historical deployment data.

Modules:
- train_pipeline: Complete training pipeline orchestration
- data_loader: Training data loading and generation utilities
- baseline_reproducer: Baseline model validation and reproduction

Usage:
    from src.training.train_pipeline import TrainingPipeline
    from src.training.data_loader import TrainingDataLoader
    from src.training.baseline_reproducer import BaselineReproducer
"""
