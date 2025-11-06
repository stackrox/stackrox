"""
Abstract base class for sample data sources.
"""

from abc import ABC, abstractmethod
from typing import Dict, Any, Iterator, Optional
import logging

logger = logging.getLogger(__name__)


class SampleStreamSource(ABC):
    """
    Abstract base class for streaming deployment data sources.

    Implementations provide raw deployment records that are then processed
    by SampleStream for feature extraction.
    """

    @abstractmethod
    def stream_samples(self,
                      filters: Optional[Dict[str, Any]] = None,
                      limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        """
        Stream raw deployment records from the data source.

        Expected record format:
        {
            "deployment": {...},      # Deployment data/protobuf as dict
            "images": [...],          # List of image data
            "alerts": [...],          # List of policy violation alerts
            "baseline_violations": [...],  # Process baseline violations
            "current_risk_score": float    # Optional: existing risk score
        }

        For Central API sources, this may also include:
        {
            "result": {
                "deployment": {...},
                "images": [...],
                "vulnerabilities": [...]
            },
            "workload_cvss": float
        }

        Args:
            filters: Optional filtering criteria (source-specific)
            limit: Optional maximum number of records to stream

        Yields:
            Raw deployment records ready for feature extraction
        """
        pass

    def close(self):
        """Clean up any resources held by this source."""
        pass
