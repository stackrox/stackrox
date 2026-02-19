# Scale Testing - AI Agent Context

For complete step-by-step workflows, **see [scale/README.md](README.md)**.

This file contains AI-specific context and gotchas when helping users with scale testing.

## Key Context for AI Agents

### Authentication Gotcha
The Central API requires authentication **even via port-forward**. API calls without auth return empty/null data instead of clear 401 errors, which looks like "no resources exist" but is actually an auth issue.

Always use: `-u admin:$(cat deploy/k8s/central-deploy/password)`

### Database is the Source of Truth
- Database name is `central_active` (not "central" or "stackrox")
- For accurate counts, query the database directly (see README)
- Don't rely on API responses for verification during scale testing

### Fake Mode Timing
When sensor initializes fake mode with large workloads:
- Logs will show `kubernetes/fake: rolebindings: 0` for ~5 minutes with xlarge.yaml (50K RBAC)
- This **looks like a hang but is normal** - it's loading RBAC resources from pebble.db
- Look for early `kubernetes/fake:` logs to confirm fake mode started, not just "Created Workload manager"

### Sensor Restart is Expected
When using `scale/dev/launch_sensor.sh`, sensor restarts once automatically (configmap mount timing). This is expected behavior, not a problem.

### Cluster Requirements Matter
The `scale/dev/` scripts patch Central to request 24 CPU total (8 for Central, 16 for Central-DB). Smaller clusters like e2-standard-8 will fail to schedule Central. The cluster requirements in the README are based on actual resource requests, not arbitrary numbers.

### Migration Testing
- Set `MAIN_IMAGE_TAG` before deploying Central to start from a specific baseline version
- Use `oc logs --tail=100000` after migration completes - you don't need to tail in real-time
- Migration timing will vary by migration type - don't make promises about performance
