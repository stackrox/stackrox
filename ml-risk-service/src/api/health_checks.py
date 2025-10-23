"""
Health check service for ML Risk Service.
Provides HTTP health endpoints and monitoring.
"""

import logging
import json
import time
from typing import Dict, Any, Optional
from datetime import datetime, timedelta
import threading
import os
import psutil

try:
    from flask import Flask, jsonify, request
    FLASK_AVAILABLE = True
except ImportError:
    FLASK_AVAILABLE = False

try:
    from prometheus_client import Counter, Histogram, Gauge, generate_latest
    PROMETHEUS_AVAILABLE = True
except ImportError:
    PROMETHEUS_AVAILABLE = False

logger = logging.getLogger(__name__)


class HealthCheckService:
    """
    Health check and monitoring service for ML Risk Service.
    Provides HTTP endpoints for health status and metrics.
    """

    def __init__(self, ml_service_impl, config: Optional[Dict[str, Any]] = None):
        self.ml_service = ml_service_impl
        self.config = config or {}
        self.start_time = time.time()

        # Health status
        self.healthy = True
        self.ready = False
        self.error_count = 0
        self.last_error = None

        # Metrics (if Prometheus available)
        if PROMETHEUS_AVAILABLE:
            self._init_prometheus_metrics()

        # Flask app (if available)
        if FLASK_AVAILABLE:
            self.app = self._create_flask_app()
        else:
            self.app = None
            logger.warning("Flask not available - HTTP health checks disabled")

        # Background monitoring
        self._monitoring_thread = None
        self._stop_monitoring = threading.Event()

    def _init_prometheus_metrics(self):
        """Initialize Prometheus metrics."""
        self.metrics = {
            'predictions_total': Counter(
                'ml_risk_predictions_total',
                'Total number of risk predictions made'
            ),
            'predictions_duration': Histogram(
                'ml_risk_prediction_duration_seconds',
                'Time spent on risk predictions'
            ),
            'training_duration': Histogram(
                'ml_risk_training_duration_seconds',
                'Time spent on model training'
            ),
            'model_loaded': Gauge(
                'ml_risk_model_loaded',
                'Whether a model is currently loaded (1=loaded, 0=not loaded)'
            ),
            'service_uptime': Gauge(
                'ml_risk_service_uptime_seconds',
                'Service uptime in seconds'
            ),
            'memory_usage': Gauge(
                'ml_risk_memory_usage_bytes',
                'Memory usage in bytes'
            ),
            'cpu_usage': Gauge(
                'ml_risk_cpu_usage_percent',
                'CPU usage percentage'
            )
        }

    def _create_flask_app(self) -> Flask:
        """Create Flask application for health endpoints."""
        app = Flask(__name__)

        @app.route('/health')
        def health():
            """Basic health check endpoint."""
            return jsonify(self.get_health_status())

        @app.route('/ready')
        def ready():
            """Readiness check endpoint."""
            status = self.get_readiness_status()
            return jsonify(status), 200 if status['ready'] else 503

        @app.route('/metrics')
        def metrics():
            """Prometheus metrics endpoint."""
            if PROMETHEUS_AVAILABLE:
                self._update_system_metrics()
                return generate_latest(), 200, {'Content-Type': 'text/plain'}
            else:
                return jsonify({'error': 'Prometheus metrics not available'}), 501

        @app.route('/status')
        def status():
            """Detailed status endpoint."""
            return jsonify(self.get_detailed_status())

        @app.route('/model/info')
        def model_info():
            """Model information endpoint."""
            try:
                info = self.ml_service.model.get_model_info()
                return jsonify(info)
            except Exception as e:
                return jsonify({'error': str(e)}), 500

        @app.route('/model/retrain', methods=['POST'])
        def retrain_model():
            """Trigger model retraining."""
            try:
                # This would trigger retraining with new data
                # For now, return a placeholder response
                return jsonify({
                    'message': 'Model retraining triggered',
                    'status': 'accepted'
                }), 202
            except Exception as e:
                return jsonify({'error': str(e)}), 500

        return app

    def get_health_status(self) -> Dict[str, Any]:
        """Get basic health status."""
        return {
            'healthy': self.healthy,
            'timestamp': datetime.now().isoformat(),
            'uptime_seconds': time.time() - self.start_time,
            'error_count': self.error_count,
            'last_error': self.last_error
        }

    def get_readiness_status(self) -> Dict[str, Any]:
        """Get readiness status (ready to serve requests)."""
        model_ready = self.ml_service.model_loaded
        self.ready = self.healthy and model_ready

        return {
            'ready': self.ready,
            'healthy': self.healthy,
            'model_loaded': model_ready,
            'timestamp': datetime.now().isoformat()
        }

    def get_detailed_status(self) -> Dict[str, Any]:
        """Get detailed service status."""
        try:
            # Get model information
            model_info = {}
            try:
                model_info = self.ml_service.model.get_model_info()
            except Exception as e:
                model_info = {'error': str(e)}

            # Get system metrics
            memory_info = psutil.virtual_memory()
            cpu_percent = psutil.cpu_percent(interval=1)

            # Get service metrics
            avg_prediction_time = (
                self.ml_service.total_prediction_time / max(self.ml_service.predictions_served, 1)
                if self.ml_service.predictions_served > 0 else 0.0
            )

            return {
                'service': {
                    'healthy': self.healthy,
                    'ready': self.ready,
                    'uptime_seconds': time.time() - self.start_time,
                    'error_count': self.error_count,
                    'last_error': self.last_error
                },
                'model': model_info,
                'metrics': {
                    'predictions_served': self.ml_service.predictions_served,
                    'avg_prediction_time_ms': avg_prediction_time,
                    'last_training_time': self.ml_service.last_training_time,
                    'training_examples_count': self.ml_service.training_examples_count
                },
                'system': {
                    'memory_usage_mb': memory_info.used / (1024 * 1024),
                    'memory_total_mb': memory_info.total / (1024 * 1024),
                    'memory_percent': memory_info.percent,
                    'cpu_percent': cpu_percent,
                    'disk_usage': self._get_disk_usage()
                },
                'timestamp': datetime.now().isoformat()
            }

        except Exception as e:
            logger.error(f"Failed to get detailed status: {e}")
            return {
                'error': str(e),
                'timestamp': datetime.now().isoformat()
            }

    def _get_disk_usage(self) -> Dict[str, float]:
        """Get disk usage information."""
        try:
            disk = psutil.disk_usage('/')
            return {
                'used_gb': disk.used / (1024**3),
                'total_gb': disk.total / (1024**3),
                'percent': (disk.used / disk.total) * 100
            }
        except Exception:
            return {}

    def _update_system_metrics(self):
        """Update system metrics for Prometheus."""
        if not PROMETHEUS_AVAILABLE:
            return

        try:
            # Update service metrics
            self.metrics['service_uptime'].set(time.time() - self.start_time)
            self.metrics['model_loaded'].set(1 if self.ml_service.model_loaded else 0)

            # Update system metrics
            memory_info = psutil.virtual_memory()
            self.metrics['memory_usage'].set(memory_info.used)

            cpu_percent = psutil.cpu_percent(interval=None)  # Non-blocking
            self.metrics['cpu_usage'].set(cpu_percent)

        except Exception as e:
            logger.warning(f"Failed to update system metrics: {e}")

    def record_prediction(self, duration_seconds: float, success: bool = True):
        """Record a prediction event."""
        if PROMETHEUS_AVAILABLE:
            self.metrics['predictions_total'].inc()
            self.metrics['predictions_duration'].observe(duration_seconds)

        if not success:
            self.error_count += 1
            self.last_error = datetime.now().isoformat()

    def record_training(self, duration_seconds: float, success: bool = True):
        """Record a training event."""
        if PROMETHEUS_AVAILABLE:
            self.metrics['training_duration'].observe(duration_seconds)

        if not success:
            self.error_count += 1
            self.last_error = datetime.now().isoformat()

    def set_health_status(self, healthy: bool, error_message: Optional[str] = None):
        """Set service health status."""
        self.healthy = healthy
        if not healthy and error_message:
            self.error_count += 1
            self.last_error = f"{datetime.now().isoformat()}: {error_message}"

    def start_monitoring(self):
        """Start background monitoring thread."""
        if self._monitoring_thread is None or not self._monitoring_thread.is_alive():
            self._stop_monitoring.clear()
            self._monitoring_thread = threading.Thread(target=self._monitoring_loop)
            self._monitoring_thread.daemon = True
            self._monitoring_thread.start()
            logger.info("Started health monitoring thread")

    def stop_monitoring(self):
        """Stop background monitoring thread."""
        if self._monitoring_thread and self._monitoring_thread.is_alive():
            self._stop_monitoring.set()
            self._monitoring_thread.join(timeout=10)
            logger.info("Stopped health monitoring thread")

    def _monitoring_loop(self):
        """Background monitoring loop."""
        while not self._stop_monitoring.wait(30):  # Check every 30 seconds
            try:
                self._perform_health_checks()
                if PROMETHEUS_AVAILABLE:
                    self._update_system_metrics()
            except Exception as e:
                logger.warning(f"Health monitoring error: {e}")

    def _perform_health_checks(self):
        """Perform periodic health checks."""
        try:
            # Check if model is still loaded and functional
            if self.ml_service.model_loaded:
                # Could perform a simple prediction test here
                pass

            # Check memory usage
            memory_info = psutil.virtual_memory()
            if memory_info.percent > 90:
                logger.warning(f"High memory usage: {memory_info.percent}%")

            # Check disk space
            disk = psutil.disk_usage('/')
            disk_percent = (disk.used / disk.total) * 100
            if disk_percent > 90:
                logger.warning(f"High disk usage: {disk_percent}%")

            # Check if service is responsive
            # Could add more sophisticated checks here

        except Exception as e:
            logger.error(f"Health check failed: {e}")
            self.set_health_status(False, str(e))

    def start_http_server(self, port: int = 8081, host: str = '0.0.0.0'):
        """Start HTTP server for health endpoints."""
        if not FLASK_AVAILABLE or not self.app:
            logger.error("Flask not available - cannot start HTTP health server")
            return

        logger.info(f"Starting health check HTTP server on {host}:{port}")

        try:
            self.app.run(host=host, port=port, debug=False, threaded=True)
        except Exception as e:
            logger.error(f"Failed to start health HTTP server: {e}")
            raise

    def run_health_server(self):
        """Run health server in separate thread."""
        if not FLASK_AVAILABLE:
            logger.warning("Flask not available - health HTTP server disabled")
            return

        health_port = self.config.get('api', {}).get('health_port', 8081)

        def run_server():
            try:
                self.start_http_server(port=health_port)
            except Exception as e:
                logger.error(f"Health server failed: {e}")

        health_thread = threading.Thread(target=run_server)
        health_thread.daemon = True
        health_thread.start()
        logger.info(f"Health server started on port {health_port}")


class HealthCheckDecorator:
    """Decorator for monitoring ML service methods."""

    def __init__(self, health_service: HealthCheckService):
        self.health_service = health_service

    def monitor_prediction(self, func):
        """Decorator to monitor prediction methods."""
        def wrapper(*args, **kwargs):
            start_time = time.time()
            success = True
            try:
                result = func(*args, **kwargs)
                return result
            except Exception as e:
                success = False
                self.health_service.set_health_status(False, f"Prediction failed: {str(e)}")
                raise
            finally:
                duration = time.time() - start_time
                self.health_service.record_prediction(duration, success)
        return wrapper

    def monitor_training(self, func):
        """Decorator to monitor training methods."""
        def wrapper(*args, **kwargs):
            start_time = time.time()
            success = True
            try:
                result = func(*args, **kwargs)
                return result
            except Exception as e:
                success = False
                self.health_service.set_health_status(False, f"Training failed: {str(e)}")
                raise
            finally:
                duration = time.time() - start_time
                self.health_service.record_training(duration, success)
        return wrapper


def create_health_service(ml_service_impl, config: Optional[Dict[str, Any]] = None) -> HealthCheckService:
    """Create and configure health check service."""
    health_service = HealthCheckService(ml_service_impl, config)

    # Start monitoring
    health_service.start_monitoring()

    # Start HTTP server if configured
    if config and config.get('api', {}).get('health_enabled', True):
        health_service.run_health_server()

    return health_service