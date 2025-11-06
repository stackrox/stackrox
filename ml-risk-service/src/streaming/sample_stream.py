"""
Unified sample streaming with feature extraction.
"""

import logging
from typing import Dict, Any, Iterator, Optional
from datetime import datetime, timezone

from src.streaming.sample_source import SampleStreamSource
from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logger = logging.getLogger(__name__)


class SampleStream:
    """
    Unified sample streaming interface with feature extraction.

    This class provides a consistent way to stream processed training/prediction
    samples from any data source (Central API, JSON files, etc.).

    It consolidates duplicate logic that was previously spread across:
    - CentralExportService._create_training_sample_from_workload()
    - TrainingDataLoader._process_deployment_record()
    """

    def __init__(self,
                 source: SampleStreamSource,
                 feature_extractor: Optional[BaselineFeatureExtractor] = None,
                 config: Optional[Dict[str, Any]] = None):
        """
        Initialize sample stream.

        Args:
            source: Data source to stream from
            feature_extractor: Feature extractor (creates one if not provided)
            config: Optional configuration
        """
        self.source = source
        self.feature_extractor = feature_extractor or BaselineFeatureExtractor()
        self.config = config or {}
        self.batch_size = self.config.get('batch_size', 100)

        # Statistics tracking
        self._stats = {
            'total_records': 0,
            'successful_samples': 0,
            'failed_samples': 0,
            'risk_scores': []
        }

    def stream(self,
              filters: Optional[Dict[str, Any]] = None,
              limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream processed samples with features extracted.

        Args:
            filters: Optional filtering criteria (passed to source)
            limit: Optional maximum number of samples to yield

        Yields:
            Training samples in standardized format:
            {
                'features': {feature_name: value, ...},
                'risk_score': float,
                'deployment_id': str,
                'deployment_name': str,
                'namespace': str,
                'cluster_id': str,
                'workload_metadata': {...}  # Source-specific metadata
            }
        """
        logger.info(f"Starting sample streaming with filters: {filters}, limit: {limit}")

        samples_yielded = 0

        try:
            for raw_record in self.source.stream_samples(filters, limit):
                self._stats['total_records'] += 1

                # Process raw record into training sample
                processed_sample = self._process_record(raw_record)

                if processed_sample:
                    self._stats['successful_samples'] += 1
                    self._stats['risk_scores'].append(processed_sample.get('risk_score', 0.0))

                    yield processed_sample
                    samples_yielded += 1

                    # Check limit
                    if limit and samples_yielded >= limit:
                        logger.info(f"Reached limit of {limit} samples")
                        break

                    # Log progress periodically
                    if samples_yielded % self.batch_size == 0:
                        logger.info(f"Streamed {samples_yielded} processed samples")
                else:
                    self._stats['failed_samples'] += 1

            # Log final summary
            self._log_final_summary(samples_yielded)

        except Exception as e:
            logger.error(f"Error during sample streaming: {e}")
            raise
        finally:
            # Clean up source
            self.source.close()

    def _process_record(self, raw_record: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Process a raw deployment record into a training sample.

        This consolidates logic from:
        - CentralExportService._create_training_sample_from_workload()
        - TrainingDataLoader._process_deployment_record()

        Args:
            raw_record: Raw deployment record from source

        Returns:
            Processed training sample or None if processing fails
        """
        try:
            # Handle two possible formats:
            # 1. Central API format with nested 'result'
            # 2. JSON file format with direct keys

            if 'result' in raw_record and raw_record['result'] is not None:
                # Central API format
                result_data = raw_record['result']
                if not isinstance(result_data, dict):
                    logger.warning(f"Invalid result type: {type(result_data)}")
                    return None

                deployment_data = result_data.get('deployment', {})
                images_data = result_data.get('images', [])
                vulnerabilities = result_data.get('vulnerabilities', [])
                alerts_data = []  # Central workload export doesn't include alerts directly
                workload_cvss = raw_record.get('workload_cvss', 0.0)
            else:
                # JSON file format or simplified format
                deployment_data = raw_record.get('deployment', {})
                images_data = raw_record.get('images', [])
                alerts_data = raw_record.get('alerts', [])
                vulnerabilities = []
                workload_cvss = 0.0

            # Validate deployment data
            if not deployment_data or not isinstance(deployment_data, dict):
                logger.warning(f"Invalid or missing deployment_data: {type(deployment_data)}")
                return None

            # Extract deployment metadata with field name fallbacks
            deployment_id = deployment_data.get('id') or deployment_data.get('deploymentId', '')
            deployment_name = (deployment_data.get('name') or
                             deployment_data.get('deploymentName') or
                             deployment_data.get('metadata', {}).get('name', ''))
            namespace = (deployment_data.get('namespace') or
                        deployment_data.get('namespaceName') or
                        deployment_data.get('metadata', {}).get('namespace', ''))
            cluster_id = (deployment_data.get('clusterId') or
                         deployment_data.get('cluster_id') or
                         deployment_data.get('clusterName', ''))

            # Get baseline violations if available
            baseline_violations = raw_record.get('baseline_violations', [])

            # Extract Central's risk score if available
            central_risk_score = deployment_data.get('riskScore')
            existing_risk_score = raw_record.get('current_risk_score')

            # Prefer explicit current_risk_score, fall back to Central's riskScore
            risk_score_to_use = existing_risk_score if existing_risk_score is not None else central_risk_score

            if risk_score_to_use is None:
                logger.debug(f"No risk score found for deployment {deployment_id}, will compute from baseline")

            # Extract features using baseline extractor
            training_sample = self.feature_extractor.create_training_sample(
                deployment_data=deployment_data,
                image_data_list=images_data,
                alert_data=alerts_data,
                baseline_violations=baseline_violations,
                risk_score=risk_score_to_use  # Use provided risk score or None to compute
            )

            # Add standardized metadata
            training_sample.update({
                'deployment_id': deployment_id,
                'deployment_name': deployment_name,
                'namespace': namespace,
                'cluster_id': cluster_id
            })

            # Add workload-specific metadata
            training_sample['workload_metadata'] = {
                'deployment_id': deployment_id,
                'deployment_name': deployment_name,
                'namespace': namespace,
                'cluster_id': cluster_id,
                'total_vulnerabilities': len(vulnerabilities),
                'workload_cvss': workload_cvss,
                'image_count': len(images_data),
                'alert_count': len(alerts_data),
                'collected_at': datetime.now(timezone.utc).isoformat()
            }

            return training_sample

        except Exception as e:
            # Get deployment info for better error reporting
            try:
                if 'result' in raw_record and isinstance(raw_record.get('result'), dict):
                    dep_id = raw_record['result'].get('deployment', {}).get('id', 'unknown')
                else:
                    dep_id = raw_record.get('deployment', {}).get('id', 'unknown')
            except:
                dep_id = 'unknown'

            logger.error(f"Failed to process record for deployment {dep_id}: {type(e).__name__}: {e}")
            logger.debug(f"Record structure: keys={list(raw_record.keys())}")
            return None

    def _log_final_summary(self, samples_yielded: int):
        """Log final statistics summary."""
        try:
            total_records = self._stats['total_records']
            successful = self._stats['successful_samples']
            failed = self._stats['failed_samples']
            risk_scores = self._stats['risk_scores']

            if risk_scores:
                import numpy as np
                avg_risk = float(np.mean(risk_scores))
                logger.info(f"Sample streaming complete: total_records={total_records} "
                          f"successful={successful} failed={failed} "
                          f"avg_risk={avg_risk:.3f}")
            else:
                logger.info(f"Sample streaming complete: total_records={total_records} "
                          f"successful={successful} failed={failed}")

            # Reset statistics for next stream
            self._stats = {
                'total_records': 0,
                'successful_samples': 0,
                'failed_samples': 0,
                'risk_scores': []
            }

        except Exception as e:
            logger.warning(f"Failed to log final summary: {e}")

    def get_stats(self) -> Dict[str, Any]:
        """Get current streaming statistics."""
        return self._stats.copy()
