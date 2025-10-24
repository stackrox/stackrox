"""
Alerting system for ML Risk Service drift detection and model monitoring.
Supports multiple alert channels: webhook, email, Slack, and logging.
"""

import logging
import json
import time
import asyncio
import smtplib
from typing import Dict, Any, List, Optional, Callable
from dataclasses import dataclass, asdict
from datetime import datetime, timezone
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart
import threading
from collections import defaultdict, deque
import requests

from .drift_detector import DriftAlert, DriftReport

logger = logging.getLogger(__name__)


@dataclass
class AlertRule:
    """Configuration for alert rules."""
    name: str
    drift_types: List[str]  # ['data_drift', 'prediction_drift', 'performance_drift']
    min_severity: str  # 'low', 'medium', 'high', 'critical'
    channels: List[str]  # ['webhook', 'email', 'slack', 'log']
    cooldown_minutes: int = 60  # Minimum time between similar alerts
    enabled: bool = True


@dataclass
class AlertChannel:
    """Base configuration for alert channels."""
    name: str
    channel_type: str  # 'webhook', 'email', 'slack', 'log'
    config: Dict[str, Any]
    enabled: bool = True


class AlertFormatter:
    """Formats alerts for different channels."""

    @staticmethod
    def format_drift_alert(alert: DriftAlert, channel_type: str = 'text') -> str:
        """Format a drift alert for a specific channel."""
        if channel_type == 'slack':
            return AlertFormatter._format_slack_alert(alert)
        elif channel_type == 'email':
            return AlertFormatter._format_email_alert(alert)
        elif channel_type == 'webhook':
            return AlertFormatter._format_webhook_alert(alert)
        else:
            return AlertFormatter._format_text_alert(alert)

    @staticmethod
    def _format_text_alert(alert: DriftAlert) -> str:
        """Format alert as plain text."""
        return (
            f"🚨 DRIFT ALERT: {alert.drift_type.upper()}\n"
            f"Severity: {alert.severity.upper()}\n"
            f"Metric: {alert.metric_name}\n"
            f"Drift Score: {alert.drift_score:.3f} (threshold: {alert.threshold:.3f})\n"
            f"Current Value: {alert.current_value:.3f}\n"
            f"Baseline Value: {alert.baseline_value:.3f}\n"
            f"Message: {alert.message}\n"
            f"Time: {alert.timestamp}"
        )

    @staticmethod
    def _format_slack_alert(alert: DriftAlert) -> Dict[str, Any]:
        """Format alert for Slack webhook."""
        severity_emoji = {
            'low': '⚠️',
            'medium': '🟡',
            'high': '🔴',
            'critical': '🚨'
        }

        severity_color = {
            'low': 'warning',
            'medium': 'warning',
            'high': 'danger',
            'critical': 'danger'
        }

        return {
            "text": f"{severity_emoji.get(alert.severity, '⚠️')} ML Model Drift Alert",
            "attachments": [
                {
                    "color": severity_color.get(alert.severity, 'warning'),
                    "fields": [
                        {
                            "title": "Drift Type",
                            "value": alert.drift_type.replace('_', ' ').title(),
                            "short": True
                        },
                        {
                            "title": "Severity",
                            "value": alert.severity.upper(),
                            "short": True
                        },
                        {
                            "title": "Metric",
                            "value": alert.metric_name,
                            "short": True
                        },
                        {
                            "title": "Drift Score",
                            "value": f"{alert.drift_score:.3f}",
                            "short": True
                        },
                        {
                            "title": "Current Value",
                            "value": f"{alert.current_value:.3f}",
                            "short": True
                        },
                        {
                            "title": "Baseline Value",
                            "value": f"{alert.baseline_value:.3f}",
                            "short": True
                        }
                    ],
                    "text": alert.message,
                    "footer": "ML Risk Service",
                    "ts": int(time.time())
                }
            ]
        }

    @staticmethod
    def _format_email_alert(alert: DriftAlert) -> Dict[str, str]:
        """Format alert for email."""
        subject = f"ML Model Drift Alert - {alert.severity.upper()} - {alert.drift_type}"

        body = f"""
        <html>
        <body>
        <h2>🚨 ML Model Drift Alert</h2>

        <table border="1" cellpadding="5" cellspacing="0">
        <tr><td><strong>Alert ID</strong></td><td>{alert.alert_id}</td></tr>
        <tr><td><strong>Drift Type</strong></td><td>{alert.drift_type.replace('_', ' ').title()}</td></tr>
        <tr><td><strong>Severity</strong></td><td><strong>{alert.severity.upper()}</strong></td></tr>
        <tr><td><strong>Metric</strong></td><td>{alert.metric_name}</td></tr>
        <tr><td><strong>Drift Score</strong></td><td>{alert.drift_score:.3f}</td></tr>
        <tr><td><strong>Threshold</strong></td><td>{alert.threshold:.3f}</td></tr>
        <tr><td><strong>Current Value</strong></td><td>{alert.current_value:.3f}</td></tr>
        <tr><td><strong>Baseline Value</strong></td><td>{alert.baseline_value:.3f}</td></tr>
        <tr><td><strong>Time</strong></td><td>{alert.timestamp}</td></tr>
        </table>

        <h3>Message</h3>
        <p>{alert.message}</p>

        <h3>Details</h3>
        <pre>{json.dumps(alert.details or {}, indent=2)}</pre>

        <hr>
        <p><em>Generated by StackRox ML Risk Service</em></p>
        </body>
        </html>
        """

        return {"subject": subject, "body": body}

    @staticmethod
    def _format_webhook_alert(alert: DriftAlert) -> Dict[str, Any]:
        """Format alert for generic webhook."""
        return {
            "alert_type": "model_drift",
            "alert": asdict(alert),
            "service": "ml-risk-service",
            "timestamp": alert.timestamp
        }

    @staticmethod
    def format_drift_report(report: DriftReport, channel_type: str = 'text') -> str:
        """Format a drift report for a specific channel."""
        if channel_type == 'slack':
            return AlertFormatter._format_slack_report(report)
        elif channel_type == 'email':
            return AlertFormatter._format_email_report(report)
        else:
            return AlertFormatter._format_text_report(report)

    @staticmethod
    def _format_text_report(report: DriftReport) -> str:
        """Format drift report as plain text."""
        return (
            f"📊 DRIFT REPORT: {report.model_id} v{report.version}\n"
            f"Overall Status: {report.overall_drift_status.upper()}\n"
            f"Overall Score: {report.overall_drift_score:.3f}\n"
            f"Data Drift: {report.data_drift_score:.3f}\n"
            f"Prediction Drift: {report.prediction_drift_score:.3f}\n"
            f"Performance Drift: {report.performance_drift_score:.3f}\n"
            f"Active Alerts: {len(report.active_alerts)}\n"
            f"Report Period: {report.report_period_hours} hours\n"
            f"Recommendations: {', '.join(report.recommendations) if report.recommendations else 'None'}\n"
            f"Generated: {report.timestamp}"
        )

    @staticmethod
    def _format_slack_report(report: DriftReport) -> Dict[str, Any]:
        """Format drift report for Slack."""
        status_emoji = {
            'no_drift': '✅',
            'low_drift': '⚠️',
            'medium_drift': '🟡',
            'high_drift': '🔴'
        }

        return {
            "text": f"{status_emoji.get(report.overall_drift_status, '📊')} ML Model Drift Report",
            "attachments": [
                {
                    "color": "good" if report.overall_drift_status == "no_drift" else "warning",
                    "fields": [
                        {
                            "title": "Model",
                            "value": f"{report.model_id} v{report.version}",
                            "short": True
                        },
                        {
                            "title": "Overall Status",
                            "value": report.overall_drift_status.replace('_', ' ').title(),
                            "short": True
                        },
                        {
                            "title": "Overall Score",
                            "value": f"{report.overall_drift_score:.3f}",
                            "short": True
                        },
                        {
                            "title": "Active Alerts",
                            "value": str(len(report.active_alerts)),
                            "short": True
                        }
                    ],
                    "text": f"Data: {report.data_drift_score:.3f} | Predictions: {report.prediction_drift_score:.3f} | Performance: {report.performance_drift_score:.3f}",
                    "footer": "ML Risk Service Drift Report",
                    "ts": int(time.time())
                }
            ]
        }

    @staticmethod
    def _format_email_report(report: DriftReport) -> Dict[str, str]:
        """Format drift report for email."""
        subject = f"ML Model Drift Report - {report.model_id} - {report.overall_drift_status.upper()}"

        body = f"""
        <html>
        <body>
        <h2>📊 ML Model Drift Report</h2>

        <table border="1" cellpadding="5" cellspacing="0">
        <tr><td><strong>Model</strong></td><td>{report.model_id} v{report.version}</td></tr>
        <tr><td><strong>Overall Status</strong></td><td><strong>{report.overall_drift_status.replace('_', ' ').title()}</strong></td></tr>
        <tr><td><strong>Overall Score</strong></td><td>{report.overall_drift_score:.3f}</td></tr>
        <tr><td><strong>Data Drift</strong></td><td>{report.data_drift_score:.3f}</td></tr>
        <tr><td><strong>Prediction Drift</strong></td><td>{report.prediction_drift_score:.3f}</td></tr>
        <tr><td><strong>Performance Drift</strong></td><td>{report.performance_drift_score:.3f}</td></tr>
        <tr><td><strong>Active Alerts</strong></td><td>{len(report.active_alerts)}</td></tr>
        <tr><td><strong>Report Period</strong></td><td>{report.report_period_hours} hours</td></tr>
        <tr><td><strong>Generated</strong></td><td>{report.timestamp}</td></tr>
        </table>

        <h3>Recommendations</h3>
        <ul>
        {''.join(f'<li>{rec}</li>' for rec in report.recommendations)}
        </ul>

        <h3>Drift Metrics</h3>
        <pre>{json.dumps(report.drift_metrics, indent=2)}</pre>

        <hr>
        <p><em>Generated by StackRox ML Risk Service</em></p>
        </body>
        </html>
        """

        return {"subject": subject, "body": body}


class AlertManager:
    """Manages alert rules, channels, and delivery."""

    def __init__(self, config: Dict[str, Any] = None):
        self.config = config or {}
        self.alert_rules: List[AlertRule] = []
        self.alert_channels: Dict[str, AlertChannel] = {}
        self.alert_history = deque(maxlen=1000)
        self.alert_cooldowns = defaultdict(float)  # Track last alert time for cooldown
        self.logger = logging.getLogger(__name__)
        self._lock = threading.RLock()

        # Setup default configuration
        self._setup_default_rules()
        self._setup_alert_channels()

    def _setup_default_rules(self):
        """Setup default alert rules."""
        default_rules = [
            AlertRule(
                name="critical_drift_alerts",
                drift_types=["data_drift", "prediction_drift", "performance_drift"],
                min_severity="critical",
                channels=["webhook", "slack", "email", "log"],
                cooldown_minutes=30
            ),
            AlertRule(
                name="high_drift_alerts",
                drift_types=["data_drift", "prediction_drift", "performance_drift"],
                min_severity="high",
                channels=["webhook", "slack", "log"],
                cooldown_minutes=60
            ),
            AlertRule(
                name="medium_drift_alerts",
                drift_types=["data_drift", "prediction_drift"],
                min_severity="medium",
                channels=["log"],
                cooldown_minutes=120
            )
        ]

        for rule in default_rules:
            self.add_alert_rule(rule)

    def _setup_alert_channels(self):
        """Setup alert channels from configuration."""
        # Webhook channel
        webhook_config = self.config.get('webhook', {})
        if webhook_config.get('enabled', False):
            self.add_alert_channel(AlertChannel(
                name="webhook",
                channel_type="webhook",
                config=webhook_config
            ))

        # Email channel
        email_config = self.config.get('email', {})
        if email_config.get('enabled', False):
            self.add_alert_channel(AlertChannel(
                name="email",
                channel_type="email",
                config=email_config
            ))

        # Slack channel
        slack_config = self.config.get('slack', {})
        if slack_config.get('enabled', False):
            self.add_alert_channel(AlertChannel(
                name="slack",
                channel_type="slack",
                config=slack_config
            ))

        # Log channel (always enabled)
        self.add_alert_channel(AlertChannel(
            name="log",
            channel_type="log",
            config={"level": "WARNING"}
        ))

    def add_alert_rule(self, rule: AlertRule):
        """Add an alert rule."""
        with self._lock:
            self.alert_rules.append(rule)
            self.logger.info(f"Added alert rule: {rule.name}")

    def add_alert_channel(self, channel: AlertChannel):
        """Add an alert channel."""
        with self._lock:
            self.alert_channels[channel.name] = channel
            self.logger.info(f"Added alert channel: {channel.name} ({channel.channel_type})")

    def should_send_alert(self, alert: DriftAlert, rule: AlertRule) -> bool:
        """Check if alert should be sent based on rules and cooldowns."""
        # Check drift type
        if alert.drift_type not in rule.drift_types:
            return False

        # Check severity
        severity_levels = {'low': 1, 'medium': 2, 'high': 3, 'critical': 4}
        if severity_levels.get(alert.severity, 0) < severity_levels.get(rule.min_severity, 0):
            return False

        # Check cooldown
        cooldown_key = f"{rule.name}:{alert.drift_type}:{alert.metric_name}"
        last_alert_time = self.alert_cooldowns.get(cooldown_key, 0)
        current_time = time.time()

        if current_time - last_alert_time < rule.cooldown_minutes * 60:
            return False

        return True

    async def send_alert(self, alert: DriftAlert):
        """Send alert through configured channels."""
        with self._lock:
            applicable_rules = [
                rule for rule in self.alert_rules
                if rule.enabled and self.should_send_alert(alert, rule)
            ]

            if not applicable_rules:
                return

            # Collect all channels to notify
            channels_to_notify = set()
            for rule in applicable_rules:
                channels_to_notify.update(rule.channels)

            # Send to each channel
            for channel_name in channels_to_notify:
                if channel_name in self.alert_channels:
                    channel = self.alert_channels[channel_name]
                    if channel.enabled:
                        try:
                            await self._send_to_channel(alert, channel)

                            # Update cooldown for applicable rules
                            current_time = time.time()
                            for rule in applicable_rules:
                                if channel_name in rule.channels:
                                    cooldown_key = f"{rule.name}:{alert.drift_type}:{alert.metric_name}"
                                    self.alert_cooldowns[cooldown_key] = current_time

                        except Exception as e:
                            self.logger.error(f"Failed to send alert to {channel_name}: {e}")

            # Store in history
            self.alert_history.append({
                'alert': alert,
                'sent_at': time.time(),
                'channels': list(channels_to_notify)
            })

    async def _send_to_channel(self, alert: DriftAlert, channel: AlertChannel):
        """Send alert to a specific channel."""
        if channel.channel_type == "webhook":
            await self._send_webhook_alert(alert, channel)
        elif channel.channel_type == "email":
            await self._send_email_alert(alert, channel)
        elif channel.channel_type == "slack":
            await self._send_slack_alert(alert, channel)
        elif channel.channel_type == "log":
            self._send_log_alert(alert, channel)

    async def _send_webhook_alert(self, alert: DriftAlert, channel: AlertChannel):
        """Send alert via webhook."""
        webhook_url = channel.config.get('url')
        if not webhook_url:
            raise ValueError("Webhook URL not configured")

        payload = AlertFormatter.format_drift_alert(alert, 'webhook')
        headers = {
            'Content-Type': 'application/json',
            **channel.config.get('headers', {})
        }

        timeout = channel.config.get('timeout', 10)

        response = requests.post(
            webhook_url,
            json=payload,
            headers=headers,
            timeout=timeout
        )
        response.raise_for_status()

        self.logger.info(f"Sent webhook alert for {alert.alert_id}")

    async def _send_email_alert(self, alert: DriftAlert, channel: AlertChannel):
        """Send alert via email."""
        smtp_config = channel.config

        # Format email
        email_content = AlertFormatter.format_drift_alert(alert, 'email')

        # Create message
        msg = MIMEMultipart('alternative')
        msg['Subject'] = email_content['subject']
        msg['From'] = smtp_config['from_email']
        msg['To'] = ', '.join(smtp_config['to_emails'])

        # Add HTML content
        html_part = MIMEText(email_content['body'], 'html')
        msg.attach(html_part)

        # Send email
        with smtplib.SMTP(smtp_config['smtp_host'], smtp_config.get('smtp_port', 587)) as server:
            if smtp_config.get('use_tls', True):
                server.starttls()

            if 'username' in smtp_config and 'password' in smtp_config:
                server.login(smtp_config['username'], smtp_config['password'])

            server.send_message(msg)

        self.logger.info(f"Sent email alert for {alert.alert_id}")

    async def _send_slack_alert(self, alert: DriftAlert, channel: AlertChannel):
        """Send alert to Slack."""
        webhook_url = channel.config.get('webhook_url')
        if not webhook_url:
            raise ValueError("Slack webhook URL not configured")

        payload = AlertFormatter.format_drift_alert(alert, 'slack')

        response = requests.post(webhook_url, json=payload, timeout=10)
        response.raise_for_status()

        self.logger.info(f"Sent Slack alert for {alert.alert_id}")

    def _send_log_alert(self, alert: DriftAlert, channel: AlertChannel):
        """Send alert to logs."""
        log_level = channel.config.get('level', 'WARNING').upper()
        message = AlertFormatter.format_drift_alert(alert, 'text')

        if log_level == 'CRITICAL':
            self.logger.critical(message)
        elif log_level == 'ERROR':
            self.logger.error(message)
        elif log_level == 'WARNING':
            self.logger.warning(message)
        else:
            self.logger.info(message)

    async def send_drift_report(self, report: DriftReport):
        """Send periodic drift report."""
        # Only send reports if there are significant findings
        if report.overall_drift_status not in ['medium_drift', 'high_drift']:
            return

        # Send to reporting channels (typically different from alert channels)
        reporting_channels = self.config.get('reporting_channels', ['log'])

        for channel_name in reporting_channels:
            if channel_name in self.alert_channels:
                channel = self.alert_channels[channel_name]
                try:
                    await self._send_report_to_channel(report, channel)
                except Exception as e:
                    self.logger.error(f"Failed to send report to {channel_name}: {e}")

    async def _send_report_to_channel(self, report: DriftReport, channel: AlertChannel):
        """Send report to a specific channel."""
        if channel.channel_type == "slack":
            payload = AlertFormatter.format_drift_report(report, 'slack')
            webhook_url = channel.config.get('webhook_url')
            if webhook_url:
                response = requests.post(webhook_url, json=payload, timeout=10)
                response.raise_for_status()

        elif channel.channel_type == "log":
            message = AlertFormatter.format_drift_report(report, 'text')
            self.logger.info(f"DRIFT REPORT:\n{message}")

    def get_alert_history(self, hours: int = 24) -> List[Dict[str, Any]]:
        """Get recent alert history."""
        cutoff_time = time.time() - (hours * 3600)
        return [
            entry for entry in self.alert_history
            if entry['sent_at'] >= cutoff_time
        ]