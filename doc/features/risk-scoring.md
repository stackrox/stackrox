# Risk Scoring

Security risk assessment and prioritization using multiplier-based scoring across deployments, images, nodes, and components.

**Primary Package**: `central/risk`

## What It Does

StackRox calculates composite risk scores by multiplying independent risk factors. The system evaluates multiple dimensions (vulnerabilities, policy violations, network exposure, baseline deviations, image age) where low scores in any dimension reduce overall risk while high scores in multiple areas compound to critical levels.

Users see prioritized deployment lists, risk breakdown by factor, and remediation guidance in the UI.

## Architecture

### Manager

The `central/risk/manager/` orchestrates risk lifecycle. Implementation in `manager_impl.go` coordinates scorers for each resource type (deployment, image, node), persists risk objects to PostgreSQL, updates resource entities with computed scores, maintains aggregate rankings (cluster, namespace), and handles component-level risk embedded in scan data.

Key methods: `ReprocessDeploymentRisk()` recalculates on changes, `CalculateRiskAndUpsertImageV2()` scores images with v2 model, `CalculateRiskAndUpsertNode()` scores Kubernetes nodes.

Execution uses elevated permissions (`riskReprocessorCtx`) for cross-namespace computation and `allAccessCtx` for scorer execution requiring varied permissions.

### Scorers

The `central/risk/scorer/` implements resource-specific calculation using multipliers.

**Deployment Scorer** (`scorer/deployment/scorer.go`): Multiplier chain ordered by importance: Violations (policy violations, max impact), ProcessBaselines (baseline deviations), ImageVulnerabilities (CVEs from images), ServiceConfig (port exposure, service type), Reachability (network accessibility), RiskyComponentCount (high-risk library count), ComponentCount (total dependencies), ImageAge (staleness).

Overall score calculation: Initialize overallScore = 1.0, iterate multipliers calling Score() with deployment and imageRisks, multiply overallScore by each multiplier's result, append results to list.

**Image Scorer** (`scorer/image/scorer.go`): Multiplier chain: Vulnerabilities (CVE analysis), RiskyComponents (known-bad libraries like Log4j, Spring4Shell), ComponentCount (dependency complexity), ImageAge (last build timestamp). Component risk separately scores each EmbeddedImageScanComponent with scores stored in component.RiskScore, not persisted separately but embedded in image scan data.

**Node Scorer** (`scorer/node/scorer.go`): Multiplier chain: Vulnerabilities (OS/kernel CVEs). Scores each EmbeddedNodeScanComponent similarly to image components.

### Multipliers

The `central/risk/multipliers/` implements pluggable risk factors returning scores between 0.0 and typically 4.0.

**Deployment Multipliers** (`multipliers/deployment/`):

- **Violations** (`violations.go`): Queries active alerts, groups by policy severity (CRITICAL/HIGH/MEDIUM/LOW), higher severity yields higher multiplier. Recent optimization in ROX-33252 queries only needed fields.
- **Process Baselines** (`process_baseline_violations.go`): Compares running processes against learned baselines, deviations indicate potential compromise, integrates with ProcessBaselineEvaluator.
- **Service Config** (`config.go`): Port exposure levels (NodePort > LoadBalancer > ClusterIP), service type risk factors, external exposure magnifies risk.
- **Reachability** (`reachability.go`): Network accessibility from outside cluster, considers ingress routes and network policies.
- **Image Multipliers** (`image_helper.go`): Aggregates image risk scores to deployment, weighted by container count.

**Image Multipliers** (`multipliers/image/`):

- **Vulnerabilities** (`vulnerabilities.go`): Processes CVEs from scan components, calculates min/max/sum of CVSS scores, normalizes via `NormalizeScore(sum, saturation=100, maxScore=4)`.
- **Risky Components** (`risky_component.go`): Binary flag for known-vulnerable library presence.
- **Component Count** (`component_count.go`): Total dependency count indicating attack surface, logarithmic scaling prevents unbounded growth.
- **Image Age** (`image_age.go`): Time since creation, older images likely missing patches, exponential decay function.

### Normalization

The `multipliers.NormalizeScore(sum, saturation, maxScore)` function converts raw metrics to [0, maxScore] range: if sum >= saturation return maxScore, else return (sum / saturation) * maxScore.

Example for vulnerability scoring: CVE CVSS scores [7.5, 9.1, 5.3, 8.2], sum = 30.1, saturation = 100, maxScore = 4, score = min(30.1 / 100 * 4, 4) = 1.204.

### Datastore

The `central/risk/datastore/` persists risk objects to PostgreSQL. Risk proto contains ID (generated as subjectID:subjectType), Subject (what is scored with ID, Type, Namespace, ClusterId), Score (overall multiplied value), and Results (individual multiplier outcomes with name, score, factors).

SAC integration: scoped access based on subject type. Deployments use cluster + namespace scope. Images are global (visible across clusters). Nodes use cluster scope.

## Data Flow

### Image Risk Update

1. Scanner completes scan → ImageV2 with scan components
2. Manager.CalculateRiskAndUpsertImageV2(image) called
3. ImageScorer.ScoreV2(ctx, image) executes
4. For each multiplier: Vulnerabilities.ScoreV2() (CVE analysis), RiskyComponents.Score() (library check), ComponentCount.Score() (dependency count), ImageAge.Score() (staleness check)
5. Multiply scores, build Risk object
6. For each component: ComponentScorer.Score() executes
7. UpsertRisk(imageRisk) persists
8. UpsertImage(image with RiskScore) updates
9. Trigger deployment risk recalculation for all deployments using image

### Deployment Risk Calculation

1. Trigger on deployment change, image update, or alert
2. Manager.ReprocessDeploymentRisk(deployment) called
3. Fetch image risks for all containers
4. DeploymentScorer.Score(ctx, deployment, imageRisks) executes
5. For each multiplier: Violations (query alerts), ProcessBaselines (check evaluator), ImageVulnerabilities (aggregate image risks), ServiceConfig (analyze exposure), Reachability (check network), image multipliers via ImageHelper
6. Multiply scores, build Risk object
7. Compare to existing risk (avoid unnecessary writes)
8. UpsertRisk(deploymentRisk) persists
9. Update namespace and cluster rankings
10. Update deployment.RiskScore field
11. UpsertDeployment(deployment) updates

### Recalculation Triggers

Events triggering deployment risk recalculation: new deployment created (initial calculation), deployment updated, image scan completed (recalculate all deployments using image), alert created/resolved (reprocess affected deployment), process baseline learned (reprocess if baseline changed).

Propagation: Image Scan → Image Risk → Deployment Risk → Namespace Rank → Cluster Rank.

## Database Schema

**Tables**: `risks` with primary key `id` (composite of subject ID and type), columns for subject_id, subject_type, subject_namespace, subject_cluster_id, score, results (JSON), indexes on subject type and score for ranking queries.

**Denormalized Fields**: Deployments.risk_score, images_v2.risk_score, nodes.risk_score for quick sorting.

Component scores embedded in scan data, not in separate table.

## Performance

**Batch Processing**: Image scans trigger recalculation potentially cascading to hundreds of deployments. Manager executes sequentially (no concurrency currently).

**Caching**: Risk objects cached in database, comparison to existing prevents redundant writes, deployment cache used for orphaned flow cleanup.

**Ranking Updates**: In-memory rankers maintain sorted score lists, incremental updates on changes, avoid full re-ranking each update.

**Query Optimization**: ROX-33252 optimized ViolationsMultiplier by querying only required alert fields instead of full objects, significantly improving performance for high-alert deployments.

## Key Concepts

**Why Multiplication**: Combines independent risk dimensions naturally, low scores in any dimension reduce overall risk, high scores in multiple areas compound to critical levels, aligns with probabilistic risk models.

**Risk vs Severity**: Severity is fixed property of CVE or policy (CVSS score), risk is contextual considering multiple factors. HIGH severity CVE in unreachable deployment has lower risk. MEDIUM severity CVE in internet-exposed deployment has higher risk.

**Component-Level Risk**: Component scores embedded in scan data for faster queries (no joins), not directly searchable, enables quick vulnerability impact assessment.

## Recent Changes

Recent work in 2024 Q4 addressed ROX-33252 (ViolationsMultiplier performance optimization), 2024 Q3 completed ImageV2 risk scoring migration using V2 image IDs, 2024 Q2 removed Active Vulnerability Management calculations and simplified the model, and 2024 Q1's ROX-31321 stopped writing component scores to separate risk table and embedded them in image/node scan data instead.

## Implementation

**Multiplier Implementations**:
- Deployment violations: `central/risk/multipliers/deployment/violations.go` (Score method)
- Process baselines: `central/risk/multipliers/deployment/process_baseline_violations.go`
- Service config: `central/risk/multipliers/deployment/config.go`, `deployment/port_exposure.go`
- Image vulnerabilities: `central/risk/multipliers/image/vulnerabilities.go` (NormalizeScore)
- Risky components: `central/risk/multipliers/image/risky_component.go`
- Component count: `central/risk/multipliers/image/component_count.go`
- Image age: `central/risk/multipliers/image/image_age.go`

**Manager**: `central/risk/manager/manager.go`, `central/risk/manager/manager_impl.go`
**Scorers**: `central/risk/scorer/deployment/scorer.go`, `central/risk/scorer/image/scorer.go`, `central/risk/scorer/node/scorer.go`
**Multipliers**: `central/risk/multipliers/deployment/`, `central/risk/multipliers/image/`, `central/risk/multipliers/component/`
**Datastore**: `central/risk/datastore/datastore_impl.go`
**Storage**: `proto/storage/risk.proto`
