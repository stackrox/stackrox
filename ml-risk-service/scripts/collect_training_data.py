#!/usr/bin/env python3
"""
Collect training data from Central sources configured in feature_config.yaml.

This script loads workload data from all enabled Central sources configured
for training and saves it to a JSON file suitable for file-based training.

Usage:
    # Collect from all enabled training Centrals
    python scripts/collect_training_data.py --output training_data.json

    # Collect with custom total limit
    python scripts/collect_training_data.py --output training_data.json --limit 5000

    # Collect with per-source limit
    python scripts/collect_training_data.py --output training_data.json --per-source-limit 1000

    # Filter by clusters
    python scripts/collect_training_data.py --output training_data.json --clusters prod-1,prod-2

    # Filter by namespaces
    python scripts/collect_training_data.py --output training_data.json --namespaces kube-system,default

    # Combine filters
    python scripts/collect_training_data.py --output training_data.json \\
        --clusters prod-1 --namespaces default --severity CRITICAL_SEVERITY
"""

import argparse
import json
import logging
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, List, Optional

# Add project root to path
project_root = Path(__file__).parent.parent
sys.path.insert(0, str(project_root))

from src.config.central_config import DataSourceConfig
from src.clients.central_export_client import CentralExportClient
from src.streaming import CentralStreamSource, SampleStream
from src.feature_extraction.baseline_features import BaselineFeatureExtractor

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class TrainingDataCollector:
    """Collects training data from Central sources."""

    def __init__(self, config_path: Optional[str] = None):
        """
        Initialize the collector.

        Args:
            config_path: Optional path to feature_config.yaml
        """
        self.config = DataSourceConfig.for_training(config_path)
        self.feature_extractor = BaselineFeatureExtractor()
        self.collected_samples = []

    def collect_from_all_sources(
        self,
        per_source_limit: Optional[int] = None,
        total_limit: Optional[int] = None,
        filters: Optional[Dict[str, Any]] = None
    ) -> List[Dict[str, Any]]:
        """
        Collect training samples from all enabled Central sources.

        Args:
            per_source_limit: Maximum samples to collect from each source
            total_limit: Maximum total samples across all sources
            filters: Optional filters to apply (clusters, namespaces, etc.)

        Returns:
            List of training samples
        """
        central_sources = self.config.get_central_sources(enabled_only=True)

        if not central_sources:
            logger.warning("No enabled Central sources found in configuration")
            return []

        logger.info(f"Found {len(central_sources)} enabled Central source(s)")

        # Get default limits from config if not specified
        if per_source_limit is None and total_limit is None:
            training_settings = self.config.source_config.get('training_settings', {})
            total_limit = training_settings.get('default_limit', 2000)
            logger.info(f"Using default total limit from config: {total_limit}")

        all_samples = []
        remaining_total = total_limit if total_limit else float('inf')

        for idx, source in enumerate(central_sources, 1):
            logger.info(f"\n{'='*60}")
            logger.info(f"Processing source {idx}/{len(central_sources)}: {source.name}")
            logger.info(f"Endpoint: {source.endpoint}")
            logger.info(f"{'='*60}")

            # Validate source configuration
            is_valid, issues = self.config.validate_source(source)
            if not is_valid:
                logger.error(f"Source validation failed: {'; '.join(issues)}")
                logger.warning(f"Skipping source: {source.name}")
                continue

            # Determine limit for this source
            if per_source_limit:
                source_limit = min(per_source_limit, int(remaining_total))
            else:
                source_limit = int(remaining_total) if remaining_total != float('inf') else None

            logger.info(f"Collecting up to {source_limit or 'unlimited'} samples from {source.name}")

            try:
                # Collect samples from this source
                samples = self._collect_from_source(source, source_limit, filters)

                logger.info(f"Collected {len(samples)} samples from {source.name}")
                all_samples.extend(samples)

                # Update remaining limit
                if total_limit:
                    remaining_total -= len(samples)
                    logger.info(f"Remaining total limit: {remaining_total}")

                    if remaining_total <= 0:
                        logger.info("Total limit reached")
                        break

            except Exception as e:
                logger.error(f"Error collecting from source {source.name}: {e}")
                logger.warning(f"Continuing with next source...")
                continue

        logger.info(f"\n{'='*60}")
        logger.info(f"Collection complete: {len(all_samples)} total samples from {idx} source(s)")
        logger.info(f"{'='*60}\n")

        return all_samples

    def _collect_from_source(
        self,
        source,
        limit: Optional[int],
        filters: Optional[Dict[str, Any]]
    ) -> List[Dict[str, Any]]:
        """
        Collect samples from a single Central source.

        Args:
            source: SourceConfig for the Central
            limit: Maximum samples to collect
            filters: Optional filters to apply

        Returns:
            List of training samples
        """
        # Create Central client
        client_config = self.config.get_client_config_for_source(source)
        auth_config = client_config.pop('authentication')
        endpoint = client_config.pop('endpoint')

        logger.info(f"Creating Central client...")
        logger.info(f"Authentication method: {auth_config['method']}")

        if auth_config['method'] == 'api_token':
            client = CentralExportClient(
                endpoint=endpoint,
                auth_token=auth_config['token'],
                config=client_config
            )
        else:  # mTLS
            client = CentralExportClient(
                endpoint=endpoint,
                auth_token='',  # Not used for mTLS
                config=client_config
            )

        # Test connection
        connection_test = client.test_connection()
        if not connection_test['success']:
            raise RuntimeError(f"Connection test failed: {connection_test['message']}")

        logger.info(f"Connected to Central: {connection_test.get('central_version', 'unknown version')}")

        # Merge source filters with provided filters
        combined_filters = self._merge_filters(source.filters, filters)

        logger.info(f"Using filters: {combined_filters}")

        # Create stream source
        stream_source = CentralStreamSource(client, client_config)

        # Create sample stream
        sample_stream = SampleStream(stream_source, self.feature_extractor, client_config)

        # Collect samples
        samples = []
        try:
            for sample in sample_stream.stream(combined_filters, limit):
                samples.append(sample)

                # Progress logging
                if len(samples) % 100 == 0:
                    logger.info(f"  Collected {len(samples)} samples...")

        finally:
            # Clean up
            client.close()

        return samples

    def _merge_filters(
        self,
        source_filters: Optional[Dict[str, Any]],
        override_filters: Optional[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """
        Merge source-specific filters with override filters.

        Args:
            source_filters: Filters from source configuration
            override_filters: Filters from command-line arguments

        Returns:
            Merged filter dictionary
        """
        merged = {}

        # Start with source filters
        if source_filters:
            merged.update(source_filters)

        # Override with command-line filters
        if override_filters:
            merged.update(override_filters)

        return merged

    def save_to_file(self, samples: List[Dict[str, Any]], output_file: str) -> None:
        """
        Save samples to JSON file.

        Args:
            samples: List of training samples
            output_file: Path to output file
        """
        output_path = Path(output_file)
        output_path.parent.mkdir(parents=True, exist_ok=True)

        logger.info(f"Saving {len(samples)} samples to {output_file}")

        with open(output_path, 'w') as f:
            json.dump(samples, f, indent=2)

        file_size_mb = output_path.stat().st_size / (1024 * 1024)
        logger.info(f"Saved successfully ({file_size_mb:.2f} MB)")

    def print_summary(self, samples: List[Dict[str, Any]]) -> None:
        """
        Print summary statistics about collected samples.

        Args:
            samples: List of training samples
        """
        if not samples:
            logger.info("No samples collected")
            return

        # Count features
        sample_features = samples[0].get('features', {})
        feature_count = len(sample_features)

        # Risk score statistics
        risk_scores = [s.get('risk_score', 0) for s in samples]
        avg_risk = sum(risk_scores) / len(risk_scores) if risk_scores else 0
        min_risk = min(risk_scores) if risk_scores else 0
        max_risk = max(risk_scores) if risk_scores else 0

        # Extract metadata for source diversity
        clusters = set()
        namespaces = set()
        for sample in samples:
            metadata = sample.get('export_metadata', {})
            clusters.add(metadata.get('cluster_id', 'unknown'))
            namespaces.add(metadata.get('namespace', 'unknown'))

        logger.info(f"\n{'='*60}")
        logger.info("COLLECTION SUMMARY")
        logger.info(f"{'='*60}")
        logger.info(f"Total samples:      {len(samples)}")
        logger.info(f"Feature count:      {feature_count}")
        logger.info(f"Risk score range:   {min_risk:.2f} - {max_risk:.2f} (avg: {avg_risk:.2f})")
        logger.info(f"Unique clusters:    {len(clusters)}")
        logger.info(f"Unique namespaces:  {len(namespaces)}")
        logger.info(f"{'='*60}\n")


def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(
        description='Collect training data from Central sources configured in feature_config.yaml',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Collect from all enabled Centrals
  python scripts/collect_training_data.py --output training_data.json

  # Collect with custom limits
  python scripts/collect_training_data.py --output training_data.json --limit 5000
  python scripts/collect_training_data.py --output training_data.json --per-source-limit 1000

  # Filter by clusters or namespaces
  python scripts/collect_training_data.py --output training_data.json --clusters prod-1,prod-2
  python scripts/collect_training_data.py --output training_data.json --namespaces kube-system

Environment Variables:
  TRAINING_CENTRAL_API_TOKEN        API token for Central authentication
  TRAINING_CENTRAL_CLIENT_CERT_PATH Client cert path for mTLS
  TRAINING_CENTRAL_CLIENT_KEY_PATH  Client key path for mTLS
  TRAINING_CENTRAL_CA_CERT_PATH     CA cert path for mTLS
        """
    )

    parser.add_argument(
        '--output', '-o',
        required=True,
        help='Output JSON file path'
    )

    parser.add_argument(
        '--config',
        help='Path to feature_config.yaml (default: auto-detect)'
    )

    parser.add_argument(
        '--limit',
        type=int,
        help='Maximum total samples to collect across all sources'
    )

    parser.add_argument(
        '--per-source-limit',
        type=int,
        help='Maximum samples to collect from each source'
    )

    # Filter arguments
    parser.add_argument(
        '--clusters',
        help='Comma-separated list of cluster IDs to filter'
    )

    parser.add_argument(
        '--namespaces',
        help='Comma-separated list of namespaces to filter'
    )

    parser.add_argument(
        '--severity',
        choices=['LOW_SEVERITY', 'MEDIUM_SEVERITY', 'HIGH_SEVERITY', 'CRITICAL_SEVERITY'],
        help='Minimum severity threshold'
    )

    parser.add_argument(
        '--include-inactive',
        action='store_true',
        help='Include inactive deployments'
    )

    args = parser.parse_args()

    # Build filters from arguments
    filters = {}
    if args.clusters:
        filters['clusters'] = [c.strip() for c in args.clusters.split(',')]
    if args.namespaces:
        filters['namespaces'] = [n.strip() for n in args.namespaces.split(',')]
    if args.severity:
        filters['severity_threshold'] = args.severity
    if args.include_inactive:
        filters['include_inactive'] = True

    try:
        # Create collector
        logger.info("Starting training data collection")
        logger.info(f"Timestamp: {datetime.now().isoformat()}")

        collector = TrainingDataCollector(config_path=args.config)

        # Collect samples
        samples = collector.collect_from_all_sources(
            per_source_limit=args.per_source_limit,
            total_limit=args.limit,
            filters=filters if filters else None
        )

        if not samples:
            logger.error("No samples collected. Check configuration and connectivity.")
            sys.exit(1)

        # Save to file
        collector.save_to_file(samples, args.output)

        # Print summary
        collector.print_summary(samples)

        logger.info(f"Success! Training data saved to: {args.output}")
        logger.info(f"\nYou can now train using:")
        logger.info(f"  curl -X POST http://localhost:8000/training/file/train?file_path={args.output}")

    except KeyboardInterrupt:
        logger.warning("\nCollection interrupted by user")
        sys.exit(130)
    except Exception as e:
        logger.error(f"Collection failed: {e}", exc_info=True)
        sys.exit(1)


if __name__ == '__main__':
    main()
