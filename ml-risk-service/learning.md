# ML Risk Service: How Training and Prediction Work

This document explains how the ML Risk Service trains models and makes predictions to rank deployment risk. No machine learning background is required.

---

## Overview: What This System Does

The ML Risk Service assigns risk scores to Kubernetes deployments based on security characteristics. It learns from historical deployment data to understand which security issues are most important for determining risk.

**Key Concept**: The model ranks deployments by risk level rather than giving absolute risk scores. This means it's designed to answer "Which deployments are riskier than others?" rather than "What is the exact risk level?"

---

## Part 1: Understanding the Algorithm

### What is RandomForest?

The system uses an algorithm called **RandomForest**, which works like a committee of experts making decisions together.

**Simple Analogy**:
Imagine you want to assess if a house is at risk of fire. You could ask 1000 safety inspectors to each create their own checklist and evaluate the house. Each inspector looks at different combinations of factors (electrical wiring, smoke detectors, flammable materials, etc.) and gives their opinion. The final risk assessment is the average of all 1000 opinions.

RandomForest works the same way:
- It creates **1000 decision trees** (the "experts")
- Each tree learns different patterns from the training data
- When predicting risk for a new deployment, all 1000 trees vote
- The final risk score is the average of all votes

### Why RandomForest?

RandomForest is particularly good for this task because:
1. **Robust**: It doesn't rely on a single decision path - if one tree makes a mistake, the other 999 can correct it
2. **Handles complexity**: It can learn complex relationships between security features without manual programming
3. **Provides explanations**: It tells us which security features are most important for risk ranking
4. **No iterative training**: Unlike neural networks, it doesn't need epochs or backpropagation - it builds all trees in one pass

---

## Part 2: Training - How the Model Learns

### Step 1: Collecting Training Data

Training data comes from StackRox Central via the `/v1/export/vuln-mgmt/workloads` API, which provides both deployment security characteristics and Central's calculated risk scores:

```
Example deployment data:
- Deployment: "nginx-frontend"
- Policy violations: 3 critical, 5 high severity
- Vulnerabilities: 12 CVEs (2 critical, 7 high, 3 medium)
- Privileged containers: Yes
- Host network access: No
- External exposure: Yes
- Process baseline violations: 2
- Central's risk score: 57.8 (ground truth)
```

Each deployment is converted into **numerical features** that the model can understand, and Central's risk score serves as the training target.

### Step 2: Feature Extraction

The system extracts security-related features from each deployment. These fall into several categories:

#### Deployment Configuration Features
- `policy_violation_score`: Weighted count of policy violations (critical violations count more)
- `privileged_container_ratio`: Percentage of containers running as privileged
- `host_network`: Whether deployment uses host networking (binary: 0 or 1)
- `host_pid`: Whether deployment accesses host PID namespace
- `host_ipc`: Whether deployment accesses host IPC namespace
- `automount_service_account_token`: Whether service account tokens are auto-mounted

#### Network Exposure Features
- `has_external_exposure`: Whether deployment is exposed outside the cluster
- `log_exposed_port_count`: Logarithmic scale of number of exposed ports

#### Vulnerability Features
- `avg_vulnerability_score`: Average vulnerability severity across all images
- `max_vulnerability_score`: Worst vulnerability found
- `avg_avg_cvss_score`: Average CVSS score of vulnerabilities
- `max_avg_cvss_score`: Maximum CVSS score found
- `sum_vulnerability_score`: Total vulnerability burden

#### Image Security Features
- `avg_risky_component_count`: Average number of risky components per image
- `max_risky_component_count`: Maximum risky components in any image
- `avg_component_count`: Average total components
- `log_avg_component_count`: Logarithmic scale (reduces impact of very large numbers)
- `avg_image_age_days`: How old the images are
- `max_image_age_days`: Age of oldest image

#### Runtime Behavior Features
- `process_baseline_violations`: Number of unexpected processes running

### Step 3: Creating Target Scores

For training, the model needs to know what risk score each deployment should have. The system uses two approaches:

**Approach 1: Central's Risk Scores (Primary)**
The primary source of training labels is Central's calculated risk scores, obtained via the `/v1/export/vuln-mgmt/workloads` API:

```python
# Central provides riskScore field for each deployment
deployment_data = {
    "id": "deployment-123",
    "name": "nginx-frontend",
    "riskScore": 57.8,  # Ground truth from Central
    ...
}
```

The model learns to reproduce and understand Central's risk assessment patterns. This approach ensures the ML model aligns with StackRox's established risk methodology while learning from actual security data.

**Approach 2: Synthetic Scoring (Fallback)**
When Central's risk scores are unavailable or incomplete, the system can calculate synthetic scores based on StackRox's risk multiplier system:

```
Synthetic Risk Score =
  Policy Violations Multiplier ×
  Process Baseline Multiplier ×
  Vulnerabilities Multiplier ×
  Service Config Multiplier ×
  Reachability Multiplier ×
  Risky Components Multiplier ×
  Component Count Multiplier ×
  Image Age Multiplier
```

Each multiplier is calculated from the deployment's features using pre-defined formulas that reproduce StackRox's risk logic. This fallback ensures training can proceed even without Central connectivity or for synthetic test data.

### Step 3.5: Understanding Baseline Features (Optional)

**What are baseline features?**
Baseline features are an optional component that reproduces StackRox's traditional risk multiplier calculations. They were originally designed to bootstrap the ML system before Central integration was available.

**When are they used?**
The `create_training_sample()` function accepts an optional `risk_score` parameter:
- **If `risk_score` is provided** (e.g., from Central's `riskScore` field): Baseline features are **not computed**, saving processing time
- **If `risk_score` is None**: Baseline features are computed and multiplied together to create a synthetic score

**Implementation detail**:
```python
# In feature extraction
def create_training_sample(
    deployment_data,
    image_data_list,
    alert_data,
    baseline_violations,
    risk_score=None  # Optional: use Central's score
):
    # Extract ML features (always done)
    features = extract_features(deployment_data, image_data_list)

    # Determine final risk score
    if risk_score is not None:
        # Use provided score (from Central)
        final_score = risk_score
        baseline_factors = None  # Skip baseline computation
    else:
        # Compute synthetic score from baseline factors
        baseline_factors = compute_baseline_factors(...)
        final_score = multiply_all_factors(baseline_factors)

    return {
        'features': features,
        'risk_score': final_score
    }
```

**Key insight**: The ML model only uses the extracted features (policy violations, vulnerabilities, etc.) for training. Whether the target score comes from Central or baseline multipliers doesn't affect the model's ability to learn - it's learning the same security patterns either way.

### Step 4: Data Preprocessing

Before training, the system preprocesses the data:

1. **Feature Scaling**: Each feature is normalized using StandardScaler
   - Calculates mean and standard deviation for each feature
   - Transforms values: `scaled_value = (original_value - mean) / std_deviation`
   - This ensures all features contribute equally regardless of their natural scale

2. **Handling Zero-Variance Features**: If a feature has identical values for all deployments, the system adds small random noise to enable meaningful scaling

3. **Data Splitting**: The data is split into two sets:
   - **Training set (80%)**: Used to build the model
   - **Validation set (20%)**: Used to test how well the model performs on unseen data

### Step 5: Building the RandomForest

The model training happens in one step:

```python
RandomForestRegressor(
    n_estimators=1000,        # Build 1000 decision trees
    max_depth=25,             # Each tree can be up to 25 levels deep
    min_samples_split=10,     # Need at least 10 samples to split a node
    min_samples_leaf=4,       # Each leaf must have at least 4 samples
    max_features="sqrt",      # Each tree uses sqrt(features) random features
    random_state=42,          # Ensures reproducible results
    n_jobs=-1                 # Use all CPU cores for parallel training
)
```

**What happens internally**:
1. For each of the 1000 trees:
   - Randomly select a subset of training samples (with replacement - called "bootstrap sampling")
   - At each decision point in the tree:
     - Randomly select sqrt(num_features) features to consider
     - Find the best feature and threshold to split the data
     - Continue splitting until reaching max_depth or min_samples_split
2. All 1000 trees are built independently and in parallel

**Example of one tree's decision path**:
```
Is vulnerability_score > 7.5?
├─ YES → Is privileged_container_ratio > 0.5?
│         ├─ YES → Risk Score = 8.2
│         └─ NO → Is external_exposure = 1?
│                  ├─ YES → Risk Score = 6.5
│                  └─ NO → Risk Score = 4.1
└─ NO → Is policy_violation_score > 3.0?
         ├─ YES → Risk Score = 5.0
         └─ NO → Risk Score = 2.3
```

### Step 6: Evaluating Performance

After training, the model is evaluated using **NDCG (Normalized Discounted Cumulative Gain)**:

**What is NDCG?**
NDCG measures how well the model ranks items. It's commonly used in search engines and recommendation systems.

- **Perfect NDCG = 1.0**: The model ranks deployments in exactly the right order
- **Random NDCG ≈ 0.5**: The model's rankings are no better than random
- **NDCG = 0.0**: The model's rankings are completely wrong

**Why NDCG instead of accuracy?**
Unlike classification (where you predict exact categories), this is a ranking problem. We care more about getting the relative order correct than the absolute scores. NDCG penalizes ranking mistakes more heavily when they occur at the top of the list (high-risk deployments).

**Typical Performance**:
- Training NDCG: 0.95-0.99 (model fits training data well)
- Validation NDCG: 0.85-0.95 (model generalizes to new deployments)

If validation NDCG is much lower than training NDCG, the model may be overfitting.

### Step 7: Feature Importance Analysis

After training, the model reports which features are most important for ranking risk:

**How Feature Importance is Calculated**:
For each feature, the model measures how much the prediction error decreases when that feature is used for splitting across all 1000 trees. Features that reduce error more are more important.

**Example Feature Importance Report**:
```
Top 5 Features:
1. policy_violation_score: 0.285 (28.5% of importance)
2. max_vulnerability_score: 0.198 (19.8%)
3. privileged_container_ratio: 0.152 (15.2%)
4. external_exposure: 0.089 (8.9%)
5. process_baseline_violations: 0.076 (7.6%)
```

This tells security teams which factors matter most for deployment risk.

---

## Part 3: Prediction - How the Model Ranks New Deployments

### Step 1: Feature Extraction

When a new deployment needs risk assessment:
1. Extract the same features used during training
2. Convert deployment metadata into numerical feature values
3. Create a feature vector with the same structure as training data

### Step 2: Feature Scaling

Apply the same scaling transformation learned during training:
```
scaled_value = (new_value - training_mean) / training_std_deviation
```

This ensures new data is on the same scale as training data.

### Step 3: Prediction with All Trees

Each of the 1000 trees makes a prediction:
1. Start at the root of the tree
2. Follow the decision path based on feature values
3. Reach a leaf node with a predicted risk score
4. Record that tree's prediction

### Step 4: Aggregation

The final risk score is the average of all 1000 predictions:
```
Final Risk Score = (Tree1_prediction + Tree2_prediction + ... + Tree1000_prediction) / 1000
```

**Example**:
- 600 trees predict risk score around 7.5
- 300 trees predict risk score around 6.8
- 100 trees predict risk score around 8.2
- Average = 7.4 (final risk score)

The distribution of predictions also provides a confidence measure.

### Step 5: Explanation Generation

To explain why a deployment received a specific risk score:

**Method 1: Global Feature Importance (Default)**
Use the feature importances learned during training to show which features contributed most to risk.

**Method 2: SHAP Values (Advanced)**
If SHAP (SHapley Additive exPlanations) is enabled:
- Calculate how much each feature value contributed to this specific prediction
- Positive SHAP values = feature increases risk
- Negative SHAP values = feature decreases risk
- Magnitude = how much influence

**Example Explanation**:
```
Deployment: nginx-frontend
Risk Score: 7.4

Top Contributing Features:
1. policy_violation_score (12.5) → +2.1 risk impact
2. max_vulnerability_score (8.9) → +1.8 risk impact
3. privileged_container_ratio (0.75) → +1.2 risk impact
4. external_exposure (1) → +0.6 risk impact
5. host_network (0) → -0.1 risk impact
```

---

## Part 4: Model Configuration

The model's behavior is controlled by configuration in `src/config/feature_config.yaml`:

### Algorithm Parameters

```yaml
model:
  algorithm: "sklearn_ranksvm"
  validation_split: 0.2          # 20% of data held for validation
  random_state: 42               # Ensures reproducible results

  sklearn_params:
    n_estimators: 1000           # Number of decision trees
    max_depth: 25                # Maximum tree depth
    min_samples_split: 10        # Minimum samples to split node
    min_samples_leaf: 4          # Minimum samples in leaf
    max_features: "sqrt"         # Features per split = sqrt(total)
    bootstrap: true              # Use bootstrap sampling
    n_jobs: -1                   # Use all CPU cores
```

**Tuning Guidance**:
- **Increase n_estimators** (e.g., 1500, 2000) for better accuracy but slower training
- **Increase max_depth** (e.g., 30) to capture more complex patterns (risk of overfitting)
- **Decrease max_depth** (e.g., 15) to prevent overfitting on small datasets
- **Increase min_samples_split** (e.g., 20) to make model more conservative
- **Change max_features** to "log2" for faster training or None for maximum accuracy

### Feature Weights

Individual features can be weighted to emphasize certain security characteristics:

```yaml
features:
  deployment:
    policy_violations:
      enabled: true
      weight: 1.0              # Highest priority

    privileged_containers:
      enabled: true
      weight: 0.9              # Very important

    host_network:
      enabled: true
      weight: 0.7              # Important

    replicas:
      enabled: true
      weight: 0.3              # Lower priority
```

---

## Part 5: Training Data Requirements

### Minimum Data Requirements

- **Minimum samples**: 50 deployments (model will warn with fewer)
- **Recommended samples**: 1000-2000 deployments for initial training
- **Maximum samples**: 10,000 deployments (configurable)
- **Feature diversity**: Training data should include deployments with varying risk profiles

### Data Quality Checks

The model performs several quality checks during training:

1. **Target Variance Check**: Ensures risk scores aren't all identical
   ```
   If all targets = 1.0 → ERROR: Cannot train ranking model
   If target variance < 0.0001 → WARNING: Low variance, limited learning
   ```

2. **Feature Variance Check**: Identifies features with no variation
   ```
   If > 50% features have zero variance → WARNING: Poor feature quality
   ```

3. **Unique Value Check**: Verifies sufficient diversity in target scores
   ```
   If only 1 unique target value → ERROR: Cannot rank
   If < 10 unique values → WARNING: Limited ranking granularity
   ```

---

## Part 6: Common Scenarios

### Scenario 1: First-Time Training

**Problem**: Need to train a model for the first time.

**Solution**: Use Central's risk scores as ground truth:
1. Connect to Central via `/v1/export/vuln-mgmt/workloads` API
2. Fetch deployment data including Central's calculated `riskScore` field
3. Extract security features from deployment metadata
4. Use Central's risk scores as training targets
5. Model learns to reproduce and understand Central's risk assessment patterns

**Fallback (if Central unavailable)**: Use synthetic scoring:
1. Calculate baseline risk factors for each deployment
2. Multiply all factors together to get synthetic score
3. Use synthetic scores as training targets
4. Model learns to reproduce StackRox's risk logic
5. Later, retrain with Central's actual scores when available

### Scenario 2: Continuous Learning (Production)

**Problem**: New deployment patterns emerge, threat landscape evolves.

**Solution**: Incremental retraining:
1. Collect new deployment data weekly/monthly
2. Combine with historical data (weighted toward recent)
3. Retrain model with expanded dataset
4. Compare new model's validation NDCG to old model
5. Deploy new model if NDCG improves or stays stable

### Scenario 3: Model Drift Detection

**Problem**: Model performance degrades over time.

**Solution**: Monitor validation metrics:
- Track validation NDCG over time
- If NDCG drops by > 5% → Trigger retraining
- Compare feature importance distributions
- Investigate if new security patterns emerged

### Scenario 4: Explaining Predictions to Users

**Problem**: Security team needs to understand why a deployment has high risk.

**Solution**: Use feature importance explanations:
1. Show top 5-10 contributing features
2. Provide feature descriptions in plain language
3. Compare to baseline/average deployment
4. Highlight unusual or extreme values

---

## Part 7: Technical Details

### Data Flow

```
1. Raw Deployment Data
   ↓
2. Feature Extraction (deployment_features.py, image_features.py)
   ↓
3. Feature Vector (numpy array: n_samples × n_features)
   ↓
4. Feature Scaling (StandardScaler)
   ↓
5. RandomForest Training/Prediction
   ↓
6. Risk Scores + Explanations
```

### Model Storage

Trained models are serialized and saved with metadata:

```
Storage Location: /app/models/models/{model_id}/v{version}/
Files:
  - model.joblib         (serialized RandomForest + scaler)
  - metadata.json        (version, metrics, timestamp)
```

**Metadata includes**:
- Model version and algorithm
- Training timestamp
- Performance metrics (train_ndcg, val_ndcg)
- Feature count and names
- Model size and checksum

### Performance Characteristics

**Training Time** (approximate):
- 100 samples: ~2-5 seconds
- 1,000 samples: ~10-30 seconds
- 10,000 samples: ~1-3 minutes

**Prediction Time**:
- Single deployment: <10ms
- Batch of 100 deployments: ~50-100ms

**Memory Usage**:
- Model size: ~5-20 MB (depends on n_estimators and max_depth)
- Peak training memory: ~500 MB for 10,000 samples

---

## Part 8: Limitations and Considerations

### What RandomForest is Good At

✅ Learning complex non-linear relationships
✅ Handling missing values gracefully
✅ Providing feature importance
✅ Avoiding overfitting (with proper parameters)
✅ Parallel training and prediction
✅ Robust to outliers

### What RandomForest is NOT Good At

❌ Extrapolation beyond training data range
❌ Handling completely new feature patterns
❌ Capturing time-series dynamics
❌ Learning from very small datasets (<50 samples)
❌ Providing probabilistic uncertainty estimates

### Important Assumptions

1. **Feature Stability**: Feature extraction logic remains consistent
2. **Risk Definition Stability**: What constitutes "high risk" doesn't change drastically
3. **Representative Training Data**: Training data covers the range of deployment types in production
4. **Feature Independence**: Features aren't perfectly correlated (multicollinearity)

---

## Appendix: Key Metrics Explained

### NDCG (Normalized Discounted Cumulative Gain)

**Formula** (simplified):
```
DCG = Σ (relevance_i / log2(i + 1))
NDCG = DCG / Ideal_DCG
```

**Interpretation**:
- Measures ranking quality on a 0-1 scale
- Values closer to 1.0 are better
- Penalizes mistakes at top of ranking more than bottom
- Industry standard for ranking evaluation

**Example**:
```
True Risk Ranking:  [10, 9, 8, 7, 6, 5, 4, 3, 2, 1]
Model Prediction:   [10, 8, 9, 7, 6, 5, 4, 3, 2, 1]
                         ↑ Swapped 8 and 9
NDCG ≈ 0.98 (small error near top)

Model Prediction:   [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
                    ↑ Completely reversed
NDCG ≈ 0.0 (terrible ranking)
```

---

## Summary

The ML Risk Service uses RandomForest to learn deployment risk patterns from StackRox Central's risk assessments:

1. **Data Collection**: Fetches deployment data and risk scores from Central's `/v1/export/vuln-mgmt/workloads` API
2. **Training**: Builds 1000 decision trees that learn to rank deployments by analyzing security features and Central's ground truth risk scores
3. **Prediction**: Averages predictions from all trees to produce robust risk scores aligned with Central's risk methodology
4. **Explanation**: Identifies which security features contribute most to risk (feature importance)
5. **Evaluation**: Uses NDCG to measure ranking quality against Central's scores

**Key Design Principle**: The model learns from Central's established risk scoring patterns, ensuring consistency with StackRox's security expertise while enabling ML-driven improvements and adaptability.

The system is designed to be transparent, explainable, and continuously improvable as new security patterns emerge.
