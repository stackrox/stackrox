---
name: test-plan
description: Generate a manual test plan for the current feature/work following StackRox testing patterns
arguments:
  - name: feature_description
    description: Brief description of what's being tested (e.g., "cluster label scoping in policies")
    required: false
---

# Test Plan Generator

Generates a manual test plan following StackRox patterns for manual feature testing.

## Usage

```bash
/test-plan "cluster label scoping in SecurityPolicy CRDs"
```

Or without arguments to be prompted:
```bash
/test-plan
```

---

## Instructions

**Step 1: Gather Context**

Get current context:
```bash
# Get current branch
git branch --show-current

# Get recent commits to understand changes
git log master..HEAD --oneline | head -10

# Get changed files to understand scope
git diff master...HEAD --name-only
```

{{#if feature_description}}
Feature being tested: {{feature_description}}
{{else}}
Infer the feature being tested from the branch name, commit messages, and changed files. Look for JIRA ticket references (ROX-XXXXX) in the branch name or commits. Check if any feature flags are referenced in the changes.
{{/if}}

**Step 2: Spawn Test Plan Generation Agent**

Use the Agent tool to generate the test plan:

```
Agent(
  description: "Generate manual test plan",
  subagent_type: "general-purpose",
  prompt: "<see prompt below>"
)
```

**Agent Prompt:**

```
You are generating a manual test plan for StackRox feature testing.

**Context provided:**
- Feature: {{feature_description}}
- Branch: <current-branch>
- JIRA: <if-provided>
- PR: <if-provided>
- Feature flags: <if-any>
- Changed files: <from git diff>

**Your task:**

Generate a comprehensive manual test plan following this EXACT structure:

---

# Manual Test Plan for <JIRA-ID or Feature Name>

**Feature:** <brief description>
**Branch:** <branch-name>
**PR:** #<number> (if exists)
**JIRA:** https://redhat.atlassian.net/browse/<JIRA-ID> (if exists)
**Feature Flags:** <list if any, or "None required">

---

## Setup

### 1. Set Environment Variables

```bash
# Get the image tag from rhacs-bot's comment on your PR
# It will look something like: 4.11.x-XXX-gABCDEF
export MAIN_IMAGE_TAG="<tag-from-pr-comment>"
echo "Using image tag: $MAIN_IMAGE_TAG"

# Set quay.io registry credentials (needed for roxie to pull images)
export REGISTRY_USERNAME="<your-quay-username>"
export REGISTRY_PASSWORD="<your-quay-password>"
```

### 2. Deploy ACS with roxie

```bash
cd ~/Workspace/roxie
# Add --features <FEATURE_FLAG_NAME> if the feature requires a feature flag
./roxie deploy --single-namespace
```

**Verify deployment:**
```bash
kubectl get pods -n stackrox

# Get Central's external IP for API access
export CENTRAL_IP=$(kubectl get svc -n stackrox central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "Central IP: $CENTRAL_IP"
```

### Prerequisites
- kubectl access to the cluster
- jq installed for JSON parsing
- Central admin password (auto-retrieved below)

### Environment Setup

```bash
# Get and export Central admin password
export PASSWORD=$(kubectl get secret -n stackrox admin-password -o jsonpath='{.data.password}' | base64 -d)

# Verify connectivity and authentication work
curl -sk -u "admin:$PASSWORD" https://$CENTRAL_IP/v1/metadata | jq
```

---

## Test Case 1: <Primary Functionality>

**Objective:** <What this test validates>

### Steps:

1. **<Action description>**
```bash
# Commands to execute the action
```

2. **Verify <what>**
```bash
# Verification commands
```
**Expected:** <Specific expected output or behavior>

3. **Check via API** (if applicable)
```bash
# Get resource details via Central API
RESOURCE_ID=$(curl -sk -u "admin:$PASSWORD" https://$CENTRAL_IP/v1/<endpoint> | jq -r '.<items>[] | select(.name == "<name>") | .id')
curl -sk -u "admin:$PASSWORD" https://$CENTRAL_IP/v1/<endpoint>/$RESOURCE_ID | jq '<relevant-fields>'
```
**Expected:** <Expected JSON output>

---

## Test Case 2: <Secondary Functionality>

**Objective:** <What this test validates>

### Steps:

[Follow same pattern as Test Case 1]

---

[Add more test cases as needed based on the feature scope]

---

## Cleanup

```bash
# Teardown the deployment (removes all resources from stackrox namespace)
cd ~/Workspace/roxie
./roxie teardown --single-namespace
```

---

## Summary

**Total Test Cases:** <N>

| Test Case | Description | Pass/Fail |
|-----------|-------------|-----------|
| 1 | <description> | |
| 2 | <description> | |
| N | <description> | |

**Result:** _____ / <N> passed

---

## Troubleshooting

### Common Issues

**1. Feature flag not enabled error**

**Symptom:** Resource rejected with message about feature flag required

**Solution:** Redeploy with feature flag enabled:
```bash
cd ~/Workspace/roxie
./roxie teardown --single-namespace
./roxie deploy --single-namespace --features <FEATURE_FLAG_NAME>
```

---

**2. curl commands return "credentials not found" or null**

**Symptom:**
- `curl` returns: `"failed to identify user with username \"admin\"`
- `jq` returns: `Cannot iterate over null (null)`

**Common Causes:**
- **CENTRAL_IP not set**: Run `export CENTRAL_IP=$(kubectl get svc -n stackrox central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}')`
- **Wrong password variable**: Use `$PASSWORD` (not `$ROX_PASSWORD`)
- **Wrong secret name**: Use `admin-password` secret (not `central-htpasswd`)

**Verification:**
```bash
# Check password is set
echo "Password length: ${#PASSWORD}"

# Test connectivity
curl -sk https://$CENTRAL_IP/v1/ping

# Test auth
curl -sk -u "admin:$PASSWORD" https://$CENTRAL_IP/v1/metadata | jq
```

---

**CRITICAL PATTERNS TO FOLLOW:**

1. **Authentication Setup:**
   - Secret name: `admin-password` (NOT `central-htpasswd`)
   - Variable name: `PASSWORD` (NOT `ROX_PASSWORD`)
   - Auth format in curl: `"admin:$PASSWORD"`
   - Always verify auth works before running tests

2. **API Command Patterns:**
   - Combine ID lookup and details fetch into single command blocks:
     ```bash
     RESOURCE_ID=$(curl -sk -u "admin:$PASSWORD" https://$CENTRAL_IP/v1/<endpoint> | jq -r '.<items>[] | select(.name == "<name>") | .id')
     curl -sk -u "admin:$PASSWORD" https://$CENTRAL_IP/v1/<endpoint>/$RESOURCE_ID | jq '<filter>'
     ```
   - Use `jq -r` for extracting single values to shell variables
   - Use `jq` (without -r) for pretty JSON output
   - Don't make users manually copy-paste IDs between commands

3. **Central API Access:**
   - Use the LoadBalancer external IP: `export CENTRAL_IP=$(kubectl get svc -n stackrox central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}')`
   - All curl commands use `https://$CENTRAL_IP/v1/...`

4. **Roxie Deployment:**
   - Deploy: `./roxie deploy --single-namespace`
   - With feature flags: `./roxie deploy --single-namespace --features <FEATURE_FLAG_NAME>`
   - Teardown: `./roxie teardown --single-namespace` (removes resources but keeps cluster)
   - Registry credentials required: Set `REGISTRY_USERNAME` and `REGISTRY_PASSWORD` env vars

5. **Test Case Structure:**
   - Start with clear objective statement
   - Use numbered steps
   - Put all commands in bash code blocks
   - Explicitly state expected outputs with "Expected:" prefix
   - Include both kubectl verification AND API verification where applicable

6. **What to Test:**
   - Review changed files to understand scope
   - Test happy path (feature working correctly)
   - Test different variations (e.g., different field types, scopes, configurations)
   - Test edge cases if applicable
   - Verify via both Kubernetes API (kubectl) and Central API (curl)
   - Test both creation/configuration AND runtime behavior if applicable

7. **Generating Test Cases:**
   - Think about natural boundaries in the feature (e.g., different object types, different scopes)
   - Each test case should validate one major aspect
   - Keep test cases focused and atomic
   - Typical count: 3-5 test cases for most features
   - Include at least one "combined" test case if feature has multiple components

8. **Expected Outputs:**
   - Be specific - show exact JSON structure expected
   - Use `jq` filters to show relevant fields only
   - Include sample values that match the test data
   - For kubectl commands, describe what you expect to see (e.g., "all conditions True")

**Generate the complete test plan now, following the structure above exactly.**
```

**Step 3: Write Test Plan to File**

Get the filename:
- If JIRA ticket: `tmp/<JIRA-ID>-test-plan.md`
- Otherwise: `tmp/<slugified-branch-name>-test-plan.md` (replace `/` with `-` in branch name)

Write the test plan returned by the agent to the file.

Display to user:
```
Test plan generated!

Written to: <filename>

Review and customize as needed. The test plan includes:
- Setup instructions with roxie deployment
- Environment configuration (password, port-forward)
- <N> test cases based on the feature scope
- Cleanup instructions
- Troubleshooting guide for common issues

Run through the test plan and update the summary table with Pass/Fail results.
```
