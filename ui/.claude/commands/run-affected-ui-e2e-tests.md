# Run Affected UI E2E Tests

Identify and execute Cypress e2e tests affected by UI code changes.

## Original Request for posterity - Claude you can ignore this section

Claude lets make you a new skill. This skill is going to be the ability to determine what Cypress e2e tests are affected by a code change in the application. This cannot easily be done statically, as the code that we build is separate from the Cypress test files, which run against a URL endpoint of a fully built and installed ACS instance.

My thinking is this:

1. When code is changed in application files (ui/apps/platform/src/*), or in other files that may affect functionality (e.g. package-lock.json) we determine which pages are affected by that code change.
2. The pages can be determined by MainPage/Body.tsx and ./routePaths.ts, and most code changes should eventually be traceable to here based on imports.
3. From the Cypress side, integration tests are found under (ui/apps/platform/cypress/integration) will helpers under (ui/apps/platform/cypress/helpers). These tests will almost always call a `visit()` function that can be used to determine the URL that is being visited.
4. The combination of the URL from Body.tsx and the URL in integration tests should give us an idea of which tests can be expected to be affected by a code change.

I would like you to create a skill that does the following:
1. When prompted, you check the diff of the current branch to some target. This might be "changes in the working directory" or "changes compared to `master`" or something else. The goal being find which UI files have changed since some point in the past.
2. Knowing these changes, efficiently devise a plan to perform the steps above to determine the most likely cypress test files that are affected by the change, that we should run.
3. Copy all of the text in this initial prompt into the skill, as a reference for our starting point.

END CLAUDE IGNORE
---

## Execution Steps

### 1. Get Changed Files
```bash
git diff --name-only origin/master...HEAD
```

Filter for UI files: `ui/apps/platform/src/**`, `ui/apps/platform/cypress/**`

### 2. Identify Affected Tests

**Early Exit Conditions:**
- If `package.json` or `package-lock.json` changed → "Recommend full test suite"
- If Cypress test files changed directly → Run those specific tests

**High-Impact Components** (flag immediately, skip expensive tracing):
- `CompoundSearchFilter` → violations, policies, vulnerabilities, compliance, clusters, network tests
- Files in `src/sagas/` → Map to test area:
  - `integrationSagas.js`, `apiTokenSagas.js` → `integrations/**`
  - `violationSagas.js` → `violations/**`
  - `policySagas.js` → `policies/**`
  - `clusterSagas.js` → `clusters/**`
  - `complianceSagas.js` → `compliance/**`, `compliance-enhanced/**`
- Files in `src/services/` → Map to corresponding test directories

**For other changes:**
- Grep for imports: `grep -r "import.*ComponentName" ui/apps/platform/src/`
- Map to routes via known structure
- Match to Cypress tests in corresponding directory

**Cypress Test Structure** (use for quick mapping):
- `cypress/integration/violations/` → `/main/violations`
- `cypress/integration/policies/` → `/main/policy-management/policies`
- `cypress/integration/vulnerabilities/` → `/main/vulnerabilities`
- `cypress/integration/compliance-enhanced/` → `/main/compliance`
- `cypress/integration/integrations/` → `/main/integrations`
- `cypress/integration/clusters/` → `/main/clusters`
- `cypress/integration/networkGraph/` → `/main/network-graph`

### 3. Execute Tests

**CRITICAL**: `npm run cypress-spec` only supports ONE spec per invocation!

**Correct format** (paths EXCLUDE `cypress/integration/` prefix):
```bash
npm run cypress-spec "subdirectory/testfile.test.js"
npm run cypress-spec "violations/**"  # for directory
```

**For multiple tests in different directories**, chain commands:
```bash
npm run cypress-spec "violations/test1.test.js" ; npm run cypress-spec "policies/test2.test.js"
```

**Execution Strategy:**
- Same directory → Use glob pattern: `"directory/**"`
- Different directories → Chain with `;`
- Run from `ui/apps/platform` directory

### 4. Report Results

Show:
- Test file(s) executed
- Pass/fail counts
- Failure details if any
- Total duration

## Quick Reference

**File Paths:**
- `ui/apps/platform/src/` - Application code
- `ui/apps/platform/cypress/integration/` - Test files
- `ui/apps/platform/src/routePaths.ts` - Route definitions

**Skip during tracing:**
- `*.test.ts`, `*.test.tsx`, `*.cy.jsx`, `*.cy.tsx`
- `__mocks__/`, `__tests__/` directories

**Common visit patterns in tests:**
- `visit(path, ...)`
- `visitFromLeftNav(title, ...)`
- `visitFromLeftNavExpandable(section, title, ...)`
- `cy.visit(url)`

## Performance Notes

With optimizations:
- Early exit: < 1s
- High-impact detection: 1-2s
- Import tracing: 2-5s
- Test execution: Variable (depends on test count)

Total analysis time: 3-8 seconds (vs 30-60s without optimizations)

## Test Execution Command:

**UPDATED**: The `npm run cypress-spec` command now supports MULTIPLE spec files in a single invocation!

**Correct format** (paths EXCLUDE `cypress/integration/` prefix - script adds it automatically):
```bash
# Single test file
npm run cypress-spec "subdirectory/testfile.test.js"

# Glob pattern
npm run cypress-spec "violations/**"

# MULTIPLE test files - comma-separated (PREFERRED)
npm run cypress-spec "violations/violationsTable.test.js,policies/policiesTable.test.js,integrations/general.test.js"

# MULTIPLE test files - space-separated (also works)
npm run cypress-spec "violations/violationsTable.test.js" "policies/policiesTable.test.js" "integrations/general.test.js"
```

**Important Notes:**
- Always run from the `ui/apps/platform` directory (current working directory)
- Paths should NOT include `cypress/integration/` prefix (script adds it automatically)
- The script prefixes each file individually: `cypress run --spec "cypress/integration/file1,cypress/integration/file2"`
- **PREFER comma-separated format** for multiple files to avoid shell escaping issues
- This runs all tests in a SINGLE Cypress instance (no reload between files) - much faster!
- Use glob patterns when all tests are in same directory
- Semicolon-chained commands (`;`) still work but are slower (full Cypress reload each time)
