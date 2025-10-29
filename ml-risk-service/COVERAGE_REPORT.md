# ML Risk Service - Test Coverage Report

**Overall Coverage: 24.8%** (824/3320 lines)

Generated: 2025-10-29

## Summary

The current test suite provides good coverage of core ML functionality (feature extraction, models) but has gaps in service layers, storage, and monitoring.

## Package-Level Coverage

| Package | Coverage | Lines Covered | Notes |
|---------|----------|---------------|-------|
| **feature_extraction** | 68.4% | 334/488 | ✅ Good coverage |
| **clients** | 59.9% | 115/192 | ✅ Good coverage |
| **models** | 46.7% | 168/360 | ⚠️ Moderate coverage |
| **config** | 23.3% | 42/180 | ⚠️ Low coverage |
| **services** | 22.2% | 165/743 | ⚠️ Low coverage |
| **storage** | 0.0% | 0/594 | ❌ No tests |
| **monitoring** | 0.0% | 0/763 | ❌ No tests |

## File-Level Details

### Feature Extraction (68.4% - Good) ✅
- `deployment_features.py`: 74.3% (110/148 lines)
- `baseline_features.py`: 71.4% (125/175 lines)
- `image_features.py`: 60.0% (99/165 lines)

**Status**: Core feature extraction is well-tested. These are the most critical components for ML accuracy.

### Models (46.7% - Moderate) ⚠️
- `ranking_model.py`: 46.7% (168/360 lines)

**Status**: Basic training and prediction are tested, but advanced features (SHAP explanations, model versioning, model storage) are not covered.

### Clients (59.9% - Good) ✅
- `central_export_client.py`: 59.5% (115/192 lines)

**Status**: Integration tests cover the happy path. Missing error handling and edge cases.

### Services (22.2% - Low) ⚠️
- `central_export_service.py`: 65.7% (165/251 lines) ✅
- `training_service.py`: 0.0% (0/267 lines) ❌
- `risk_service.py`: 0.0% (0/144 lines) ❌
- `model_service.py`: 0.0% (0/81 lines) ❌

**Status**: Only `central_export_service.py` is tested (via integration tests). Other services have no test coverage.

### Storage (0% - No Tests) ❌
- `model_storage.py`: 0.0% (0/594 lines)

**Status**: Model persistence, versioning, and retrieval are completely untested.

### Monitoring (0% - No Tests) ❌
- `alerting.py`: 0.0% (0/293 lines)
- `drift_detector.py`: 0.0% (0/310 lines)
- `health_checker.py`: 0.0% (0/160 lines)

**Status**: Monitoring, alerting, and drift detection are completely untested.

### Config (23.3% - Low) ⚠️
- `central_config.py`: 23.3% (42/180 lines)

**Status**: Basic configuration is tested, but many config options are not exercised.

## Test Suite Breakdown

### Current Tests (10 total)

**Unit Tests (7)**:
- ✅ `test_synthetic_data_generation_and_prediction` - End-to-end synthetic workflow
- ✅ `test_prediction_output_format` - Prediction result structure
- ✅ `test_model_training_with_synthetic_data` - Training metrics
- ✅ `test_synthetic_data_structure` - Data generation validation
- ✅ `test_feature_extraction_from_synthetic_data` - Feature extraction
- ✅ `test_predictions_consistency` - Deterministic predictions
- ✅ `test_batch_predictions` - Batch processing

**Integration Tests (3)**:
- ✅ `test_central_connection` - Central API connectivity
- ✅ `test_train_and_predict_with_central_data` - Full ML pipeline with real data
- ✅ `test_feature_extraction_from_central_data` - Feature extraction from Central

## Coverage Gaps & Recommendations

### Critical Gaps ❌

1. **Storage Layer (0%)**
   - No tests for model persistence
   - No tests for model versioning
   - No tests for model retrieval
   - **Risk**: Model corruption, version conflicts in production

2. **Monitoring Layer (0%)**
   - No tests for drift detection
   - No tests for alerting
   - No tests for health checks
   - **Risk**: Production issues go undetected

3. **Service Layer (22.2%)**
   - `training_service.py`: No tests
   - `risk_service.py`: No tests
   - `model_service.py`: No tests
   - **Risk**: Service failures, incorrect orchestration

### Moderate Gaps ⚠️

4. **Model Advanced Features (46.7%)**
   - SHAP explanations not tested
   - Model versioning edge cases
   - Model loading/saving
   - **Risk**: Incorrect explanations, version conflicts

5. **Configuration (23.3%)**
   - Many config options untested
   - Environment variable handling
   - **Risk**: Misconfiguration in production

6. **Error Handling**
   - Network failures not tested
   - Invalid data handling
   - Resource exhaustion
   - **Risk**: Poor error messages, crashes

### Recommendations

**Priority 1 (Critical for Production)**:
1. Add storage layer tests (model save/load/version)
2. Add service layer tests (training, risk, model services)
3. Add error handling tests

**Priority 2 (Important for Reliability)**:
1. Add monitoring tests (drift detection, health checks)
2. Increase model coverage (explanations, versioning)
3. Add integration tests for full workflows

**Priority 3 (Nice to Have)**:
1. Increase config coverage
2. Add performance tests
3. Add stress tests

## Test Coverage Goals

| Component | Current | Target | Priority |
|-----------|---------|--------|----------|
| feature_extraction | 68.4% | 80% | P3 |
| models | 46.7% | 75% | P2 |
| clients | 59.9% | 70% | P3 |
| services | 22.2% | 70% | P1 |
| storage | 0.0% | 70% | P1 |
| monitoring | 0.0% | 60% | P2 |
| config | 23.3% | 60% | P2 |
| **Overall** | **24.8%** | **70%** | - |

## Running Tests

```bash
# Run all tests
make test

# Run with coverage report
uv run pytest tests/ --cov=src --cov-report=term-missing --cov-report=html

# Run unit tests only
uv run pytest tests/ -m unit

# Run integration tests only
uv run pytest tests/ -m integration

# Run specific test file
uv run pytest tests/test_synthetic_predictions.py -v
```

## Notes

- Integration tests require `CENTRAL_API_TOKEN` environment variable
- Coverage report available in `htmlcov/index.html`
- Some untested code may be defensive/error handling that's hard to trigger
