"""
Model health monitoring and validation system for ML Risk Service.
Provides comprehensive health checks, performance monitoring, and validation.
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

logger = logging.getLogger(__name__)


@dataclass
class HealthCheckResult:
    """Result of a health check."""
    check_name: str
    status: str  # healthy, warning, critical, error
    score: float  # 0.0 to 1.0
    message: str
    details: Dict[str, Any] = None
    timestamp: str = None

    def __post_init__(self):
        if self.timestamp is None:
            self.timestamp = datetime.now(timezone.utc).isoformat()


@dataclass
class ModelHealthReport:
    """Comprehensive model health report."""
    model_id: str
    version: str
    overall_status: str  # healthy, warning, critical, error
    overall_score: float  # 0.0 to 1.0
    health_checks: List[HealthCheckResult]
    performance_metrics: Dict[str, float]
    recommendations: List[str]
    timestamp: str

    @classmethod
    def create(cls, model_id: str, version: str, health_checks: List[HealthCheckResult],
               performance_metrics: Dict[str, float] = None) -> 'ModelHealthReport':
        """Create health report from check results."""
        if not health_checks:
            return cls(
                model_id=model_id,
                version=version,
                overall_status="error",
                overall_score=0.0,
                health_checks=[],
                performance_metrics=performance_metrics or {},
                recommendations=["No health checks performed"],
                timestamp=datetime.now(timezone.utc).isoformat()
            )

        # Calculate overall status and score
        scores = [check.score for check in health_checks]
        overall_score = statistics.mean(scores)

        # Determine overall status
        critical_checks = [c for c in health_checks if c.status == "critical"]
        warning_checks = [c for c in health_checks if c.status == "warning"]

        if critical_checks:
            overall_status = "critical"
        elif warning_checks:
            overall_status = "warning"
        else:
            overall_status = "healthy"

        # Generate recommendations
        recommendations = []
        for check in health_checks:
            if check.status in ["warning", "critical"]:
                recommendations.append(f"{check.check_name}: {check.message}")

        return cls(
            model_id=model_id,
            version=version,
            overall_status=overall_status,
            overall_score=overall_score,
            health_checks=health_checks,
            performance_metrics=performance_metrics or {},
            recommendations=recommendations,
            timestamp=datetime.now(timezone.utc).isoformat()
        )


class HealthCheck(ABC):
    """Abstract base class for health checks."""

    def __init__(self, name: str, weight: float = 1.0):
        self.name = name
        self.weight = weight
        self.logger = logging.getLogger(f"{__name__}.{self.__class__.__name__}")

    @abstractmethod
    def check(self, model, recent_predictions: List[Dict], **kwargs) -> HealthCheckResult:
        """Perform the health check."""
        pass


class PerformanceRegressionCheck(HealthCheck):
    """Check for performance regression compared to baseline."""

    def __init__(self, baseline_metrics: Dict[str, float],
                 regression_threshold: float = 0.05):
        super().__init__("Performance Regression")
        self.baseline_metrics = baseline_metrics
        self.regression_threshold = regression_threshold

    def check(self, model, recent_predictions: List[Dict], **kwargs) -> HealthCheckResult:
        """Check for performance regression."""
        try:
            current_metrics = kwargs.get('current_metrics', {})

            if not current_metrics:
                return HealthCheckResult(
                    check_name=self.name,
                    status="error",
                    score=0.0,
                    message="No current metrics available for comparison"
                )

            regressions = []
            total_regression = 0.0
            metric_count = 0

            for metric, baseline_value in self.baseline_metrics.items():
                if metric in current_metrics:
                    current_value = current_metrics[metric]

                    # Calculate relative change
                    if baseline_value != 0:
                        change = (current_value - baseline_value) / baseline_value

                        # For metrics where higher is better (NDCG, AUC)
                        if metric in ['ndcg', 'auc', 'precision', 'recall', 'f1']:
                            if change < -self.regression_threshold:
                                regressions.append(f"{metric}: {change:.2%} decline")
                            total_regression += max(0, -change)

                        # For metrics where lower is better (loss, error rate)
                        elif metric in ['loss', 'error_rate', 'latency']:
                            if change > self.regression_threshold:
                                regressions.append(f"{metric}: {change:.2%} increase")
                            total_regression += max(0, change)

                        metric_count += 1

            if metric_count == 0:
                return HealthCheckResult(
                    check_name=self.name,
                    status="error",
                    score=0.0,
                    message="No comparable metrics found"
                )

            avg_regression = total_regression / metric_count
            score = max(0.0, 1.0 - avg_regression)

            if regressions:
                status = "critical" if avg_regression > self.regression_threshold * 2 else "warning"
                message = f"Performance regression detected: {'; '.join(regressions)}"
            else:
                status = "healthy"
                message = "No significant performance regression detected"

            return HealthCheckResult(
                check_name=self.name,
                status=status,
                score=score,
                message=message,
                details={
                    'baseline_metrics': self.baseline_metrics,
                    'current_metrics': current_metrics,
                    'regressions': regressions
                }
            )

        except Exception as e:
            return HealthCheckResult(
                check_name=self.name,
                status="error",
                score=0.0,
                message=f"Error during performance regression check: {e}"
            )


class PredictionQualityCheck(HealthCheck):
    """Check prediction quality and consistency."""

    def __init__(self, score_range: Tuple[float, float] = (0.0, 10.0),
                 consistency_threshold: float = 0.1):
        super().__init__("Prediction Quality")
        self.score_range = score_range
        self.consistency_threshold = consistency_threshold

    def check(self, model, recent_predictions: List[Dict], **kwargs) -> HealthCheckResult:
        """Check prediction quality."""
        try:
            if not recent_predictions:
                return HealthCheckResult(
                    check_name=self.name,
                    status="warning",
                    score=0.5,
                    message="No recent predictions available for quality check"
                )

            scores = [pred.get('risk_score', 0.0) for pred in recent_predictions]

            # Check for valid score range
            out_of_range = [s for s in scores if not (self.score_range[0] <= s <= self.score_range[1])]
            if out_of_range:
                return HealthCheckResult(
                    check_name=self.name,
                    status="critical",
                    score=0.0,
                    message=f"Predictions out of valid range: {len(out_of_range)}/{len(scores)}",
                    details={'out_of_range_scores': out_of_range[:10]}  # Limit for logging
                )

            # Check for NaN or infinite values
            invalid_scores = [s for s in scores if not np.isfinite(s)]
            if invalid_scores:
                return HealthCheckResult(
                    check_name=self.name,
                    status="critical",
                    score=0.0,
                    message=f"Invalid predictions (NaN/Inf): {len(invalid_scores)}/{len(scores)}"
                )

            # Check consistency (standard deviation)
            if len(scores) > 1:
                std_dev = np.std(scores)
                mean_score = np.mean(scores)
                coefficient_variation = std_dev / mean_score if mean_score != 0 else float('inf')

                if coefficient_variation > self.consistency_threshold:
                    status = "warning"
                    score = 0.7
                    message = f"High prediction variance (CV: {coefficient_variation:.3f})"
                else:
                    status = "healthy"
                    score = 1.0
                    message = f"Prediction quality good (CV: {coefficient_variation:.3f})"
            else:
                status = "healthy"
                score = 1.0
                message = "Single prediction - quality check passed"

            return HealthCheckResult(
                check_name=self.name,
                status=status,
                score=score,
                message=message,
                details={
                    'prediction_count': len(scores),
                    'mean_score': float(np.mean(scores)),
                    'std_dev': float(np.std(scores)),
                    'score_range': [float(min(scores)), float(max(scores))]
                }
            )

        except Exception as e:
            return HealthCheckResult(
                check_name=self.name,
                status="error",
                score=0.0,
                message=f"Error during prediction quality check: {e}"
            )


class LatencyCheck(HealthCheck):
    """Check prediction latency and response times."""

    def __init__(self, max_latency_ms: float = 500.0,
                 percentile_threshold: float = 100.0):
        super().__init__("Response Latency")
        self.max_latency_ms = max_latency_ms
        self.percentile_threshold = percentile_threshold

    def check(self, model, recent_predictions: List[Dict], **kwargs) -> HealthCheckResult:
        """Check prediction latency."""
        try:
            latencies = []
            for pred in recent_predictions:
                if 'latency_ms' in pred:
                    latencies.append(pred['latency_ms'])

            if not latencies:
                return HealthCheckResult(
                    check_name=self.name,
                    status="warning",
                    score=0.5,
                    message="No latency data available"
                )

            mean_latency = np.mean(latencies)
            p95_latency = np.percentile(latencies, 95)
            max_latency = np.max(latencies)

            # Determine status based on latency
            if max_latency > self.max_latency_ms * 2:
                status = "critical"
                score = 0.0
                message = f"Severe latency issues (max: {max_latency:.1f}ms)"
            elif p95_latency > self.max_latency_ms:
                status = "warning"
                score = 0.5
                message = f"High latency detected (P95: {p95_latency:.1f}ms)"
            elif mean_latency > self.percentile_threshold:
                status = "warning"
                score = 0.7
                message = f"Elevated average latency ({mean_latency:.1f}ms)"
            else:
                status = "healthy"
                score = 1.0
                message = f"Latency within acceptable range (avg: {mean_latency:.1f}ms)"

            return HealthCheckResult(
                check_name=self.name,
                status=status,
                score=score,
                message=message,
                details={
                    'mean_latency_ms': float(mean_latency),
                    'p95_latency_ms': float(p95_latency),
                    'max_latency_ms': float(max_latency),
                    'sample_count': len(latencies)
                }
            )

        except Exception as e:
            return HealthCheckResult(
                check_name=self.name,
                status="error",
                score=0.0,
                message=f"Error during latency check: {e}"
            )


class ModelStabilityCheck(HealthCheck):
    """Check model stability and consistency over time."""

    def __init__(self, stability_window: int = 100):
        super().__init__("Model Stability")
        self.stability_window = stability_window

    def check(self, model, recent_predictions: List[Dict], **kwargs) -> HealthCheckResult:
        """Check model stability."""
        try:
            if len(recent_predictions) < 10:
                return HealthCheckResult(
                    check_name=self.name,
                    status="warning",
                    score=0.5,
                    message="Insufficient data for stability analysis"
                )

            # Take the most recent predictions up to the window size
            recent_scores = [pred.get('risk_score', 0.0)
                           for pred in recent_predictions[-self.stability_window:]]

            # Calculate stability metrics
            if len(recent_scores) < 2:
                return HealthCheckResult(
                    check_name=self.name,
                    status="warning",
                    score=0.5,
                    message="Need at least 2 predictions for stability check"
                )

            # Calculate moving variance
            window_size = min(20, len(recent_scores) // 2)
            if window_size < 2:
                window_size = 2

            variances = []
            for i in range(window_size, len(recent_scores)):
                window_scores = recent_scores[i-window_size:i]
                variances.append(np.var(window_scores))

            if not variances:
                return HealthCheckResult(
                    check_name=self.name,
                    status="warning",
                    score=0.5,
                    message="Unable to calculate stability metrics"
                )

            avg_variance = np.mean(variances)
            max_variance = np.max(variances)
            overall_variance = np.var(recent_scores)

            # Determine stability status
            variance_threshold_warning = 0.1
            variance_threshold_critical = 0.5

            if max_variance > variance_threshold_critical:
                status = "critical"
                score = 0.0
                message = f"High instability detected (max variance: {max_variance:.3f})"
            elif avg_variance > variance_threshold_warning:
                status = "warning"
                score = 0.6
                message = f"Moderate instability (avg variance: {avg_variance:.3f})"
            else:
                status = "healthy"
                score = 1.0
                message = f"Model predictions stable (variance: {overall_variance:.3f})"

            return HealthCheckResult(
                check_name=self.name,
                status=status,
                score=score,
                message=message,
                details={
                    'overall_variance': float(overall_variance),
                    'avg_variance': float(avg_variance),
                    'max_variance': float(max_variance),
                    'prediction_count': len(recent_scores)
                }
            )

        except Exception as e:
            return HealthCheckResult(
                check_name=self.name,
                status="error",
                score=0.0,
                message=f"Error during stability check: {e}"
            )


class ModelHealthChecker:
    """Main health checker that orchestrates all health checks."""

    def __init__(self):
        self.health_checks: List[HealthCheck] = []
        self.prediction_history = deque(maxlen=1000)  # Keep last 1000 predictions
        self.health_history = deque(maxlen=100)  # Keep last 100 health reports
        self.logger = logging.getLogger(__name__)
        self._lock = threading.RLock()

    def add_health_check(self, health_check: HealthCheck):
        """Add a health check to the checker."""
        with self._lock:
            self.health_checks.append(health_check)
            self.logger.info(f"Added health check: {health_check.name}")

    def setup_default_checks(self, baseline_metrics: Dict[str, float] = None):
        """Setup default health checks."""
        if baseline_metrics:
            self.add_health_check(PerformanceRegressionCheck(baseline_metrics))

        self.add_health_check(PredictionQualityCheck())
        self.add_health_check(LatencyCheck())
        self.add_health_check(ModelStabilityCheck())

    def record_prediction(self, prediction_data: Dict[str, Any]):
        """Record a prediction for health monitoring."""
        with self._lock:
            prediction_data['timestamp'] = time.time()
            self.prediction_history.append(prediction_data)

    def run_health_checks(self, model, model_id: str, version: str,
                         current_metrics: Dict[str, float] = None) -> ModelHealthReport:
        """Run all health checks and generate a report."""
        with self._lock:
            if not self.health_checks:
                self.logger.warning("No health checks configured")
                return ModelHealthReport.create(model_id, version, [])

            # Get recent predictions
            recent_predictions = list(self.prediction_history)

            # Run all health checks
            check_results = []
            for health_check in self.health_checks:
                try:
                    result = health_check.check(
                        model,
                        recent_predictions,
                        current_metrics=current_metrics
                    )
                    check_results.append(result)
                    self.logger.debug(f"Health check '{health_check.name}': {result.status}")
                except Exception as e:
                    self.logger.error(f"Health check '{health_check.name}' failed: {e}")
                    check_results.append(HealthCheckResult(
                        check_name=health_check.name,
                        status="error",
                        score=0.0,
                        message=f"Check failed with error: {e}"
                    ))

            # Create health report
            report = ModelHealthReport.create(model_id, version, check_results, current_metrics)

            # Store in history
            self.health_history.append(report)

            self.logger.info(f"Health check completed for {model_id} v{version}: "
                           f"{report.overall_status} (score: {report.overall_score:.2f})")

            return report

    def get_health_trends(self, hours: int = 24) -> Dict[str, Any]:
        """Get health trends over time."""
        with self._lock:
            cutoff_time = datetime.now(timezone.utc) - timedelta(hours=hours)
            cutoff_timestamp = cutoff_time.isoformat()

            recent_reports = [
                report for report in self.health_history
                if report.timestamp >= cutoff_timestamp
            ]

            if not recent_reports:
                return {
                    'period_hours': hours,
                    'report_count': 0,
                    'trend': 'no_data'
                }

            scores = [report.overall_score for report in recent_reports]
            statuses = [report.overall_status for report in recent_reports]

            # Calculate trend
            if len(scores) >= 2:
                # Simple linear trend
                x = list(range(len(scores)))
                trend_slope = np.polyfit(x, scores, 1)[0]

                if trend_slope > 0.01:
                    trend = 'improving'
                elif trend_slope < -0.01:
                    trend = 'degrading'
                else:
                    trend = 'stable'
            else:
                trend = 'insufficient_data'

            status_counts = {
                'healthy': statuses.count('healthy'),
                'warning': statuses.count('warning'),
                'critical': statuses.count('critical'),
                'error': statuses.count('error')
            }

            return {
                'period_hours': hours,
                'report_count': len(recent_reports),
                'trend': trend,
                'avg_score': float(np.mean(scores)),
                'min_score': float(np.min(scores)),
                'max_score': float(np.max(scores)),
                'status_distribution': status_counts,
                'latest_status': recent_reports[-1].overall_status if recent_reports else None
            }

    def clear_history(self):
        """Clear prediction and health history."""
        with self._lock:
            self.prediction_history.clear()
            self.health_history.clear()
            self.logger.info("Cleared health monitoring history")