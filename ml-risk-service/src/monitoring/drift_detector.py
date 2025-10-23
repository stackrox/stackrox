"""
Model drift detection and monitoring system for ML Risk Service.
Detects data drift, prediction drift, and performance drift over time.
"""

import logging
import time
import numpy as np
import pandas as pd
from typing import Dict, Any, List, Optional, Tuple, Union
from dataclasses import dataclass, asdict
from datetime import datetime, timezone, timedelta
from abc import ABC, abstractmethod
import threading
from collections import deque
import statistics
import json

logger = logging.getLogger(__name__)


@dataclass
class DriftAlert:
    """Represents a drift detection alert."""
    alert_id: str
    drift_type: str  # data_drift, prediction_drift, performance_drift
    severity: str  # low, medium, high, critical
    metric_name: str
    current_value: float
    baseline_value: float
    drift_score: float
    threshold: float
    message: str
    timestamp: str
    details: Dict[str, Any] = None

    def __post_init__(self):
        if self.timestamp is None:
            self.timestamp = datetime.now(timezone.utc).isoformat()


@dataclass
class DriftReport:
    """Comprehensive drift detection report."""
    model_id: str
    version: str
    report_period_hours: int
    overall_drift_status: str  # no_drift, low_drift, medium_drift, high_drift
    overall_drift_score: float  # 0.0 to 1.0
    data_drift_score: float
    prediction_drift_score: float
    performance_drift_score: float
    active_alerts: List[DriftAlert]
    drift_metrics: Dict[str, float]
    recommendations: List[str]
    timestamp: str

    @classmethod
    def create(cls, model_id: str, version: str, period_hours: int,
               data_drift: float, prediction_drift: float, performance_drift: float,
               alerts: List[DriftAlert]) -> 'DriftReport':
        """Create a drift report from component scores."""

        # Calculate overall drift score (weighted average)
        overall_score = (
            data_drift * 0.4 +
            prediction_drift * 0.4 +
            performance_drift * 0.2
        )

        # Determine overall status
        if overall_score >= 0.8:
            status = "high_drift"
        elif overall_score >= 0.5:
            status = "medium_drift"
        elif overall_score >= 0.2:
            status = "low_drift"
        else:
            status = "no_drift"

        # Generate recommendations
        recommendations = []
        if data_drift > 0.5:
            recommendations.append("Consider retraining model due to data drift")
        if prediction_drift > 0.5:
            recommendations.append("Monitor prediction quality closely")
        if performance_drift > 0.5:
            recommendations.append("Investigate performance degradation")
        if overall_score > 0.6:
            recommendations.append("Urgent: Model requires attention")

        return cls(
            model_id=model_id,
            version=version,
            report_period_hours=period_hours,
            overall_drift_status=status,
            overall_drift_score=overall_score,
            data_drift_score=data_drift,
            prediction_drift_score=prediction_drift,
            performance_drift_score=performance_drift,
            active_alerts=alerts,
            drift_metrics={
                'data_drift': data_drift,
                'prediction_drift': prediction_drift,
                'performance_drift': performance_drift,
                'overall_drift': overall_score
            },
            recommendations=recommendations,
            timestamp=datetime.now(timezone.utc).isoformat()
        )


class DriftDetector(ABC):
    """Abstract base class for drift detectors."""

    def __init__(self, name: str, threshold: float = 0.5):
        self.name = name
        self.threshold = threshold
        self.logger = logging.getLogger(f"{__name__}.{self.__class__.__name__}")

    @abstractmethod
    def detect_drift(self, baseline_data: np.ndarray, current_data: np.ndarray) -> Tuple[float, Dict[str, Any]]:
        """Detect drift between baseline and current data.

        Returns:
            Tuple of (drift_score, details) where drift_score is 0.0-1.0
        """
        pass


class KolmogorovSmirnovDetector(DriftDetector):
    """Detects drift using Kolmogorov-Smirnov test."""

    def __init__(self, threshold: float = 0.05):
        super().__init__("Kolmogorov-Smirnov", threshold)

    def detect_drift(self, baseline_data: np.ndarray, current_data: np.ndarray) -> Tuple[float, Dict[str, Any]]:
        """Detect drift using KS test."""
        try:
            from scipy.stats import ks_2samp

            # Perform KS test
            statistic, p_value = ks_2samp(baseline_data, current_data)

            # Convert p-value to drift score (lower p-value = higher drift)
            drift_score = 1.0 - p_value

            details = {
                'ks_statistic': float(statistic),
                'p_value': float(p_value),
                'baseline_size': len(baseline_data),
                'current_size': len(current_data),
                'significant_drift': p_value < self.threshold
            }

            return drift_score, details

        except ImportError:
            # Fallback to simple statistical comparison
            return self._statistical_fallback(baseline_data, current_data)
        except Exception as e:
            self.logger.error(f"KS test failed: {e}")
            return 0.0, {'error': str(e)}

    def _statistical_fallback(self, baseline_data: np.ndarray, current_data: np.ndarray) -> Tuple[float, Dict[str, Any]]:
        """Fallback drift detection using statistical measures."""
        baseline_mean = np.mean(baseline_data)
        current_mean = np.mean(current_data)
        baseline_std = np.std(baseline_data)
        current_std = np.std(current_data)

        # Calculate normalized differences
        mean_diff = abs(current_mean - baseline_mean) / (baseline_std + 1e-8)
        std_diff = abs(current_std - baseline_std) / (baseline_std + 1e-8)

        # Combined drift score
        drift_score = min(1.0, (mean_diff + std_diff) / 2.0)

        details = {
            'baseline_mean': float(baseline_mean),
            'current_mean': float(current_mean),
            'baseline_std': float(baseline_std),
            'current_std': float(current_std),
            'mean_diff_normalized': float(mean_diff),
            'std_diff_normalized': float(std_diff)
        }

        return drift_score, details


class PopulationStabilityDetector(DriftDetector):
    """Detects drift using Population Stability Index (PSI)."""

    def __init__(self, threshold: float = 0.2, n_bins: int = 10):
        super().__init__("Population Stability Index", threshold)
        self.n_bins = n_bins

    def detect_drift(self, baseline_data: np.ndarray, current_data: np.ndarray) -> Tuple[float, Dict[str, Any]]:
        """Detect drift using PSI."""
        try:
            # Create bins based on baseline data
            bin_edges = np.histogram_bin_edges(baseline_data, bins=self.n_bins)

            # Calculate distributions
            baseline_hist, _ = np.histogram(baseline_data, bins=bin_edges)
            current_hist, _ = np.histogram(current_data, bins=bin_edges)

            # Convert to proportions
            baseline_props = baseline_hist / len(baseline_data)
            current_props = current_hist / len(current_data)

            # Add small epsilon to avoid log(0)
            epsilon = 1e-8
            baseline_props = baseline_props + epsilon
            current_props = current_props + epsilon

            # Calculate PSI
            psi = np.sum((current_props - baseline_props) * np.log(current_props / baseline_props))

            # Normalize PSI to 0-1 scale (PSI > 0.2 is high drift)
            drift_score = min(1.0, psi / 0.5)

            details = {
                'psi_value': float(psi),
                'n_bins': self.n_bins,
                'baseline_distribution': baseline_props.tolist(),
                'current_distribution': current_props.tolist(),
                'high_drift': psi > self.threshold
            }

            return drift_score, details

        except Exception as e:
            self.logger.error(f"PSI calculation failed: {e}")
            return 0.0, {'error': str(e)}


class ModelDriftMonitor:
    """Main drift monitoring system that orchestrates all drift detection."""

    def __init__(self, config: Dict[str, Any] = None):
        self.config = config or {}
        self.drift_detectors: List[DriftDetector] = []
        self.baseline_data = {}  # Store baseline feature distributions
        self.prediction_history = deque(maxlen=10000)  # Store predictions for drift analysis
        self.performance_history = deque(maxlen=1000)  # Store performance metrics
        self.drift_alerts = deque(maxlen=100)  # Store recent alerts
        self.drift_reports = deque(maxlen=50)  # Store recent reports
        self.logger = logging.getLogger(__name__)
        self._lock = threading.RLock()

        # Configure drift detection thresholds
        self.data_drift_threshold = self.config.get('data_drift_threshold', 0.3)
        self.prediction_drift_threshold = self.config.get('prediction_drift_threshold', 0.3)
        self.performance_drift_threshold = self.config.get('performance_drift_threshold', 0.2)

        # Setup default detectors
        self._setup_default_detectors()

    def _setup_default_detectors(self):
        """Setup default drift detectors."""
        self.drift_detectors = [
            KolmogorovSmirnovDetector(threshold=0.05),
            PopulationStabilityDetector(threshold=0.2)
        ]
        self.logger.info(f"Configured {len(self.drift_detectors)} drift detectors")

    def add_drift_detector(self, detector: DriftDetector):
        """Add a custom drift detector."""
        with self._lock:
            self.drift_detectors.append(detector)
            self.logger.info(f"Added drift detector: {detector.name}")

    def set_baseline_data(self, model_id: str, version: str, feature_data: Dict[str, np.ndarray]):
        """Set baseline feature distributions for a model."""
        with self._lock:
            baseline_key = f"{model_id}:{version}"
            self.baseline_data[baseline_key] = {
                'features': feature_data.copy(),
                'timestamp': time.time(),
                'model_id': model_id,
                'version': version
            }
            self.logger.info(f"Set baseline data for {baseline_key} with {len(feature_data)} features")

    def record_prediction(self, model_id: str, version: str, features: Dict[str, float],
                         prediction: float, timestamp: float = None):
        """Record a prediction for drift monitoring."""
        with self._lock:
            prediction_data = {
                'model_id': model_id,
                'version': version,
                'features': features.copy(),
                'prediction': prediction,
                'timestamp': timestamp or time.time()
            }
            self.prediction_history.append(prediction_data)

    def record_performance_metric(self, model_id: str, version: str,
                                 metric_name: str, value: float, timestamp: float = None):
        """Record a performance metric for drift monitoring."""
        with self._lock:
            metric_data = {
                'model_id': model_id,
                'version': version,
                'metric_name': metric_name,
                'value': value,
                'timestamp': timestamp or time.time()
            }
            self.performance_history.append(metric_data)

    def detect_data_drift(self, model_id: str, version: str,
                         recent_hours: int = 24) -> Tuple[float, List[DriftAlert]]:
        """Detect data drift for a specific model."""
        with self._lock:
            baseline_key = f"{model_id}:{version}"
            if baseline_key not in self.baseline_data:
                return 0.0, []

            baseline_features = self.baseline_data[baseline_key]['features']

            # Get recent predictions for the model
            cutoff_time = time.time() - (recent_hours * 3600)
            recent_predictions = [
                p for p in self.prediction_history
                if p['model_id'] == model_id and p['timestamp'] >= cutoff_time
            ]

            if len(recent_predictions) < 10:
                return 0.0, []

            alerts = []
            feature_drift_scores = []

            # Check drift for each feature
            for feature_name, baseline_data in baseline_features.items():
                current_data = np.array([
                    p['features'].get(feature_name, 0.0)
                    for p in recent_predictions
                ])

                if len(current_data) == 0:
                    continue

                # Run all detectors on this feature
                detector_scores = []
                for detector in self.drift_detectors:
                    try:
                        drift_score, details = detector.detect_drift(baseline_data, current_data)
                        detector_scores.append(drift_score)

                        # Create alert if drift is significant
                        if drift_score > self.data_drift_threshold:
                            alert = DriftAlert(
                                alert_id=f"data_drift_{model_id}_{feature_name}_{int(time.time())}",
                                drift_type="data_drift",
                                severity="high" if drift_score > 0.7 else "medium",
                                metric_name=feature_name,
                                current_value=float(np.mean(current_data)),
                                baseline_value=float(np.mean(baseline_data)),
                                drift_score=drift_score,
                                threshold=self.data_drift_threshold,
                                message=f"Data drift detected in feature {feature_name} using {detector.name}",
                                details=details
                            )
                            alerts.append(alert)

                    except Exception as e:
                        self.logger.error(f"Drift detection failed for feature {feature_name} with {detector.name}: {e}")

                # Average drift score across detectors for this feature
                if detector_scores:
                    feature_drift_scores.append(statistics.mean(detector_scores))

            # Overall data drift score
            overall_drift_score = statistics.mean(feature_drift_scores) if feature_drift_scores else 0.0

            return overall_drift_score, alerts

    def detect_prediction_drift(self, model_id: str, version: str,
                               recent_hours: int = 24) -> Tuple[float, List[DriftAlert]]:
        """Detect prediction drift for a specific model."""
        with self._lock:
            baseline_key = f"{model_id}:{version}"
            if baseline_key not in self.baseline_data:
                return 0.0, []

            # Get baseline predictions (if available)
            baseline_timestamp = self.baseline_data[baseline_key]['timestamp']
            baseline_window = 7 * 24 * 3600  # 7 days window for baseline

            baseline_predictions = [
                p['prediction'] for p in self.prediction_history
                if (p['model_id'] == model_id and
                    baseline_timestamp <= p['timestamp'] <= baseline_timestamp + baseline_window)
            ]

            # Get recent predictions
            cutoff_time = time.time() - (recent_hours * 3600)
            recent_predictions = [
                p['prediction'] for p in self.prediction_history
                if p['model_id'] == model_id and p['timestamp'] >= cutoff_time
            ]

            if len(baseline_predictions) < 10 or len(recent_predictions) < 10:
                return 0.0, []

            baseline_data = np.array(baseline_predictions)
            current_data = np.array(recent_predictions)

            alerts = []
            detector_scores = []

            # Run drift detectors on predictions
            for detector in self.drift_detectors:
                try:
                    drift_score, details = detector.detect_drift(baseline_data, current_data)
                    detector_scores.append(drift_score)

                    if drift_score > self.prediction_drift_threshold:
                        alert = DriftAlert(
                            alert_id=f"prediction_drift_{model_id}_{int(time.time())}",
                            drift_type="prediction_drift",
                            severity="high" if drift_score > 0.7 else "medium",
                            metric_name="prediction_distribution",
                            current_value=float(np.mean(current_data)),
                            baseline_value=float(np.mean(baseline_data)),
                            drift_score=drift_score,
                            threshold=self.prediction_drift_threshold,
                            message=f"Prediction drift detected using {detector.name}",
                            details=details
                        )
                        alerts.append(alert)

                except Exception as e:
                    self.logger.error(f"Prediction drift detection failed with {detector.name}: {e}")

            overall_drift_score = statistics.mean(detector_scores) if detector_scores else 0.0
            return overall_drift_score, alerts

    def detect_performance_drift(self, model_id: str, version: str,
                                recent_hours: int = 24) -> Tuple[float, List[DriftAlert]]:
        """Detect performance drift for a specific model."""
        with self._lock:
            # Get baseline performance (historical average)
            baseline_cutoff = time.time() - (30 * 24 * 3600)  # 30 days ago
            baseline_metrics = {}

            for metric in self.performance_history:
                if (metric['model_id'] == model_id and
                    metric['timestamp'] >= baseline_cutoff and
                    metric['timestamp'] <= baseline_cutoff + (7 * 24 * 3600)):  # 7-day baseline window

                    metric_name = metric['metric_name']
                    if metric_name not in baseline_metrics:
                        baseline_metrics[metric_name] = []
                    baseline_metrics[metric_name].append(metric['value'])

            # Get recent performance
            recent_cutoff = time.time() - (recent_hours * 3600)
            recent_metrics = {}

            for metric in self.performance_history:
                if (metric['model_id'] == model_id and
                    metric['timestamp'] >= recent_cutoff):

                    metric_name = metric['metric_name']
                    if metric_name not in recent_metrics:
                        recent_metrics[metric_name] = []
                    recent_metrics[metric_name].append(metric['value'])

            alerts = []
            drift_scores = []

            # Compare each metric
            for metric_name in baseline_metrics:
                if metric_name not in recent_metrics:
                    continue

                baseline_values = np.array(baseline_metrics[metric_name])
                recent_values = np.array(recent_metrics[metric_name])

                if len(baseline_values) < 5 or len(recent_values) < 5:
                    continue

                # Calculate drift using simple statistical comparison
                baseline_mean = np.mean(baseline_values)
                recent_mean = np.mean(recent_values)
                baseline_std = np.std(baseline_values)

                # Normalized difference
                if baseline_std > 0:
                    drift_score = abs(recent_mean - baseline_mean) / baseline_std
                    drift_score = min(1.0, drift_score / 3.0)  # Normalize to 0-1
                else:
                    drift_score = 0.0

                drift_scores.append(drift_score)

                if drift_score > self.performance_drift_threshold:
                    alert = DriftAlert(
                        alert_id=f"performance_drift_{model_id}_{metric_name}_{int(time.time())}",
                        drift_type="performance_drift",
                        severity="high" if drift_score > 0.7 else "medium",
                        metric_name=metric_name,
                        current_value=float(recent_mean),
                        baseline_value=float(baseline_mean),
                        drift_score=drift_score,
                        threshold=self.performance_drift_threshold,
                        message=f"Performance drift detected in {metric_name}",
                        details={
                            'baseline_std': float(baseline_std),
                            'baseline_count': len(baseline_values),
                            'recent_count': len(recent_values)
                        }
                    )
                    alerts.append(alert)

            overall_drift_score = statistics.mean(drift_scores) if drift_scores else 0.0
            return overall_drift_score, alerts

    def generate_drift_report(self, model_id: str, version: str,
                             report_period_hours: int = 24) -> DriftReport:
        """Generate a comprehensive drift report for a model."""
        with self._lock:
            # Detect all types of drift
            data_drift_score, data_alerts = self.detect_data_drift(model_id, version, report_period_hours)
            prediction_drift_score, prediction_alerts = self.detect_prediction_drift(model_id, version, report_period_hours)
            performance_drift_score, performance_alerts = self.detect_performance_drift(model_id, version, report_period_hours)

            # Combine all alerts
            all_alerts = data_alerts + prediction_alerts + performance_alerts

            # Store alerts
            for alert in all_alerts:
                self.drift_alerts.append(alert)

            # Create report
            report = DriftReport.create(
                model_id=model_id,
                version=version,
                period_hours=report_period_hours,
                data_drift=data_drift_score,
                prediction_drift=prediction_drift_score,
                performance_drift=performance_drift_score,
                alerts=all_alerts
            )

            # Store report
            self.drift_reports.append(report)

            self.logger.info(f"Generated drift report for {model_id} v{version}: "
                           f"{report.overall_drift_status} (score: {report.overall_drift_score:.3f})")

            return report

    def get_active_alerts(self, model_id: str = None, severity: str = None) -> List[DriftAlert]:
        """Get active drift alerts with optional filtering."""
        with self._lock:
            alerts = list(self.drift_alerts)

            if model_id:
                # Extract model_id from alert_id (format: drift_type_model_id_...)
                alerts = [a for a in alerts if model_id in a.alert_id]

            if severity:
                alerts = [a for a in alerts if a.severity == severity]

            return alerts

    def clear_history(self):
        """Clear all drift monitoring history."""
        with self._lock:
            self.prediction_history.clear()
            self.performance_history.clear()
            self.drift_alerts.clear()
            self.drift_reports.clear()
            self.baseline_data.clear()
            self.logger.info("Cleared drift monitoring history")