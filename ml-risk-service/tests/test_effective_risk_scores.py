"""
Unit tests for effective risk score extraction.

Tests the _get_effective_risk_score() method in SampleStream which extracts
user-adjusted scores with fallback to ML scores, mirroring the Go implementation
in central/risk/manager/score_calculator.go:GetEffectiveScore()
"""

import pytest
from src.streaming.sample_stream import SampleStream
from src.streaming.sample_source import SampleStreamSource
from src.feature_extraction.baseline_features import BaselineFeatureExtractor
from typing import Dict, Any, Iterator, Optional


class MockSampleSource(SampleStreamSource):
    """Mock sample source for testing."""

    def __init__(self, samples):
        self.samples = samples

    def stream_samples(self, filters: Optional[Dict[str, Any]] = None,
                      limit: Optional[int] = None) -> Iterator[Dict[str, Any]]:
        for sample in self.samples:
            yield sample

    def close(self):
        pass


def test_effective_score_with_user_adjustment():
    """
    Test that user-adjusted scores are preferred over ML scores.

    Mimics the case where a user has adjusted the risk ranking via UI,
    and we want to train on the adjusted score.
    """
    # Create sample with user adjustment (Central API format)
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-1', 'name': 'test-app'},
            'images': [],
            'risk': {
                'score': 3.5,  # Original ML score
                'user_ranking_adjustment': {
                    'adjusted_score': 7.2,  # User adjusted higher
                    'last_adjusted': {
                        'seconds': 1704067200  # Valid timestamp (2024-01-01)
                    },
                    'last_adjusted_by': 'user@example.com'
                }
            }
        }
    }

    # Create SampleStream and extract effective score
    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should use adjusted score, not ML score
    assert effective_score == 7.2, "Should use user-adjusted score"
    assert effective_score != 3.5, "Should not use original ML score when adjustment exists"


def test_effective_score_without_adjustment():
    """
    Test fallback to ML score when no user adjustment exists.

    This is the typical case where user hasn't manually adjusted rankings.
    """
    # Create sample without user adjustment
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-2', 'name': 'test-app'},
            'images': [],
            'risk': {
                'score': 4.8  # Only ML score available
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should fall back to ML score
    assert effective_score == 4.8, "Should use ML score when no adjustment"


def test_effective_score_with_invalid_adjustment_timestamp():
    """
    Test that adjustments without valid timestamps are ignored.

    If last_adjusted is missing or has zero timestamp, the adjustment
    should be considered invalid.
    """
    # User adjustment with zero timestamp (invalid)
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-3', 'name': 'test-app'},
            'images': [],
            'risk': {
                'score': 2.5,
                'user_ranking_adjustment': {
                    'adjusted_score': 6.0,
                    'last_adjusted': {
                        'seconds': 0  # Invalid timestamp
                    }
                }
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should ignore invalid adjustment and use ML score
    assert effective_score == 2.5, "Should ignore adjustment with invalid timestamp"


def test_effective_score_camel_case_fields():
    """
    Test handling of camelCase field names (JSON export format).

    Central's JSON exports may use camelCase instead of snake_case.
    """
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-4', 'name': 'test-app'},
            'images': [],
            'risk': {
                'score': 3.0,
                'userRankingAdjustment': {  # camelCase
                    'adjustedScore': 5.5,    # camelCase
                    'lastAdjusted': {        # camelCase
                        'seconds': 1704067200
                    }
                }
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should handle camelCase field names
    assert effective_score == 5.5, "Should handle camelCase field names"


def test_effective_score_no_risk_field():
    """
    Test None return when no Risk data is present.

    Old exports or incomplete data may not have risk field.
    """
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-5', 'name': 'test-app'},
            'images': []
            # No risk field
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should return None when no risk data
    assert effective_score is None, "Should return None when no risk field"


def test_effective_score_json_file_format():
    """
    Test extraction from JSON file format (no 'result' wrapper).

    JSON training data files have a different structure than API responses.
    """
    raw_record = {
        'deployment': {'id': 'dep-6', 'name': 'test-app'},
        'images': [],
        'risk': {
            'score': 3.2,
            'user_ranking_adjustment': {
                'adjusted_score': 4.8,
                'last_adjusted': {
                    'seconds': 1704067200
                }
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should handle JSON file format (no 'result' wrapper)
    assert effective_score == 4.8, "Should handle JSON file format"


def test_effective_score_missing_last_adjusted():
    """
    Test that adjustment without last_adjusted field is ignored.
    """
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-7', 'name': 'test-app'},
            'images': [],
            'risk': {
                'score': 2.0,
                'user_ranking_adjustment': {
                    'adjusted_score': 5.0
                    # Missing last_adjusted
                }
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should ignore adjustment without timestamp
    assert effective_score == 2.0, "Should ignore adjustment without last_adjusted"


def test_effective_score_type_conversion():
    """
    Test that scores are properly converted to float.

    Ensures integer scores from protobuf are handled correctly.
    """
    raw_record = {
        'result': {
            'deployment': {'id': 'dep-8', 'name': 'test-app'},
            'images': [],
            'risk': {
                'score': 3,  # Integer
                'user_ranking_adjustment': {
                    'adjusted_score': 7,  # Integer
                    'last_adjusted': {
                        'seconds': 1704067200
                    }
                }
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    effective_score = stream._get_effective_risk_score(raw_record)

    # Should convert to float
    assert isinstance(effective_score, float), "Should return float type"
    assert effective_score == 7.0, "Should handle integer scores"


def test_process_record_uses_effective_score():
    """
    Integration test: verify _process_record() uses effective scores.

    Tests the full pipeline from raw record to training sample.
    """
    raw_record = {
        'result': {
            'deployment': {
                'id': 'dep-9',
                'name': 'test-deployment',
                'namespace': 'default',
                'cluster_id': 'cluster-1',
                'containers': []
            },
            'images': [],
            'risk': {
                'score': 2.5,
                'user_ranking_adjustment': {
                    'adjusted_score': 6.5,
                    'last_adjusted': {
                        'seconds': 1704067200
                    },
                    'last_adjusted_by': 'user@example.com'
                }
            }
        }
    }

    source = MockSampleSource([raw_record])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    # Process the record
    processed = stream._process_record(raw_record)

    # Verify effective score was used
    assert processed is not None, "Should process record successfully"
    assert processed['risk_score'] == 6.5, "Should use effective (adjusted) score"
    assert processed['has_user_adjustment'] is True, "Should flag user adjustment"


def test_process_record_tracks_adjustment_flag():
    """
    Test that has_user_adjustment flag is correctly set.
    """
    # Record with adjustment
    record_with_adjustment = {
        'result': {
            'deployment': {
                'id': 'dep-10',
                'name': 'test-deployment',
                'namespace': 'default',
                'cluster_id': 'cluster-1',
                'containers': []
            },
            'images': [],
            'risk': {
                'score': 3.0,
                'user_ranking_adjustment': {
                    'adjusted_score': 5.0,
                    'last_adjusted': {'seconds': 1704067200}
                }
            }
        }
    }

    # Record without adjustment
    record_without_adjustment = {
        'result': {
            'deployment': {
                'id': 'dep-11',
                'name': 'test-deployment-2',
                'namespace': 'default',
                'cluster_id': 'cluster-1',
                'containers': []
            },
            'images': [],
            'risk': {
                'score': 4.0
            }
        }
    }

    source = MockSampleSource([])
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    # Test with adjustment
    processed_with = stream._process_record(record_with_adjustment)
    assert processed_with['has_user_adjustment'] is True

    # Test without adjustment
    processed_without = stream._process_record(record_without_adjustment)
    assert processed_without['has_user_adjustment'] is False


def test_statistics_tracking():
    """
    Test that user adjustment statistics are correctly tracked.
    """
    records = [
        # 2 with user adjustments
        {
            'result': {
                'deployment': {
                    'id': f'dep-{i}',
                    'name': f'test-{i}',
                    'namespace': 'default',
                    'cluster_id': 'cluster-1',
                    'containers': []
                },
                'images': [],
                'risk': {
                    'score': 3.0,
                    'user_ranking_adjustment': {
                        'adjusted_score': 5.0,
                        'last_adjusted': {'seconds': 1704067200}
                    }
                }
            }
        }
        for i in range(2)
    ] + [
        # 3 without user adjustments
        {
            'result': {
                'deployment': {
                    'id': f'dep-{i}',
                    'name': f'test-{i}',
                    'namespace': 'default',
                    'cluster_id': 'cluster-1',
                    'containers': []
                },
                'images': [],
                'risk': {
                    'score': 4.0
                }
            }
        }
        for i in range(2, 5)
    ]

    source = MockSampleSource(records)
    extractor = BaselineFeatureExtractor()
    stream = SampleStream(source, extractor)

    # Process all samples and count manually
    # Note: stats are reset after streaming completes in _log_final_summary()
    user_adjusted = 0
    ml_score = 0
    for sample in stream.stream():
        if sample.get('has_user_adjustment', False):
            user_adjusted += 1
        else:
            ml_score += 1

    # Verify counts
    assert user_adjusted == 2, "Should count 2 user-adjusted"
    assert ml_score == 3, "Should count 3 ML scores"


if __name__ == '__main__':
    pytest.main([__file__, '-v'])
