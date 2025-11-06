"""
JSON file sample stream source.
"""

import json
import logging
from pathlib import Path
from typing import Dict, Any, Iterator, Optional

from src.streaming.sample_source import SampleStreamSource

logger = logging.getLogger(__name__)


class JSONFileStreamSource(SampleStreamSource):
    """
    Stream deployment samples from a JSON file.

    Expected JSON file format:
    {
        "deployments": [
            {
                "deployment": {...},
                "images": [...],
                "alerts": [...],
                "baseline_violations": [...],
                "current_risk_score": 2.5  # Optional
            },
            ...
        ],
        "metadata": {...}  # Optional
    }

    This implementation provides memory-efficient streaming by loading
    the deployments array but yielding items one at a time.
    """

    def __init__(self, file_path: str):
        """
        Initialize JSON file stream source.

        Args:
            file_path: Path to JSON file containing deployment data
        """
        self.file_path = Path(file_path)

        if not self.file_path.exists():
            raise FileNotFoundError(f"Training data file not found: {file_path}")

        if not self.file_path.is_file():
            raise ValueError(f"Path is not a file: {file_path}")

    def stream_samples(self,
                      filters: Optional[Dict[str, Any]] = None,
                      limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream deployment records from JSON file.

        Args:
            filters: Not used for file source (could be used for filtering in future)
            limit: Maximum number of records to stream

        Yields:
            Raw deployment records in JSON file format:
            {
                "deployment": {...},
                "images": [...],
                "alerts": [...],
                "baseline_violations": [...],
                "current_risk_score": float  # Optional
            }
        """
        logger.info(f"Starting JSON file streaming from {self.file_path} (limit: {limit})")

        try:
            with open(self.file_path, 'r') as f:
                data = json.load(f)

            # Validate file structure
            if not isinstance(data, dict):
                raise ValueError(f"Expected JSON object at root, got {type(data)}")

            deployments = data.get('deployments', [])

            if not isinstance(deployments, list):
                raise ValueError(f"Expected 'deployments' to be a list, got {type(deployments)}")

            logger.info(f"Found {len(deployments)} deployment records in file")

            # Stream deployments one at a time
            records_yielded = 0
            for i, deployment_record in enumerate(deployments):
                if not isinstance(deployment_record, dict):
                    logger.warning(f"Skipping invalid record at index {i}: expected dict, got {type(deployment_record)}")
                    continue

                yield deployment_record
                records_yielded += 1

                if limit and records_yielded >= limit:
                    logger.info(f"Reached limit of {limit} records")
                    break

            logger.info(f"JSON file streaming completed: {records_yielded} records")

        except json.JSONDecodeError as e:
            logger.error(f"Failed to parse JSON file {self.file_path}: {e}")
            raise ValueError(f"Invalid JSON in file {self.file_path}: {e}")
        except Exception as e:
            logger.error(f"Error reading file {self.file_path}: {e}")
            raise

    def close(self):
        """Clean up resources (no resources to clean for file source)."""
        pass


class JSONLinesStreamSource(SampleStreamSource):
    """
    Stream deployment samples from a JSON Lines (.jsonl) file.

    Each line in the file should be a single deployment record in JSON format.
    This provides true streaming without loading the entire file into memory.

    Expected file format (one JSON object per line):
    {"deployment": {...}, "images": [...], "alerts": [...]}
    {"deployment": {...}, "images": [...], "alerts": [...]}
    ...
    """

    def __init__(self, file_path: str):
        """
        Initialize JSON Lines stream source.

        Args:
            file_path: Path to .jsonl file containing deployment data
        """
        self.file_path = Path(file_path)

        if not self.file_path.exists():
            raise FileNotFoundError(f"Training data file not found: {file_path}")

        if not self.file_path.is_file():
            raise ValueError(f"Path is not a file: {file_path}")

    def stream_samples(self,
                      filters: Optional[Dict[str, Any]] = None,
                      limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream deployment records from JSON Lines file.

        Args:
            filters: Not used for file source
            limit: Maximum number of records to stream

        Yields:
            Raw deployment records, one per line
        """
        logger.info(f"Starting JSON Lines streaming from {self.file_path} (limit: {limit})")

        records_yielded = 0
        line_number = 0

        try:
            with open(self.file_path, 'r') as f:
                for line in f:
                    line_number += 1
                    line = line.strip()

                    if not line:
                        continue  # Skip empty lines

                    try:
                        record = json.loads(line)
                        yield record
                        records_yielded += 1

                        if limit and records_yielded >= limit:
                            logger.info(f"Reached limit of {limit} records")
                            break

                    except json.JSONDecodeError as e:
                        logger.warning(f"Skipping invalid JSON at line {line_number}: {e}")
                        continue

            logger.info(f"JSON Lines streaming completed: {records_yielded} records from {line_number} lines")

        except Exception as e:
            logger.error(f"Error reading file {self.file_path}: {e}")
            raise

    def close(self):
        """Clean up resources (no resources to clean for file source)."""
        pass
