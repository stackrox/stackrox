# StackRox Architecture Overview for Frontend Engineers

## What is StackRox?

StackRox is a **Kubernetes security platform** that helps organizations protect their containerized applications. It monitors clusters for security threats, scans images for vulnerabilities, and enforces security policies.

## High-Level Architecture

StackRox uses a **hub-and-spoke architecture** where one Central instance manages security for multiple Kubernetes clusters.

```
┌─────────────────────────────────────┐
│     Management Cluster              │
│  ┌──────────┐  ┌─────────┐         │
│  │ Central  │  │ Scanner │         │
│  │   +UI    │  │         │         │
│  │   +DB    │  └─────────┘         │
│  └──────────┘                       │
└─────────────────────────────────────┘
           │
           │ (secure connections)
           │
    ┌──────┴──────┬───────────┐
    │             │           │
┌───▼────┐   ┌────▼───┐  ┌───▼────┐
│Cluster1│   │Cluster2│  │Cluster3│
│┌──────┐│   │┌──────┐│  │┌──────┐│
││Sensor││   ││Sensor││  ││Sensor││
│└──────┘│   │└──────┘│  │└──────┘│
└────────┘   └────────┘  └────────┘
```

## Core Components

### 1. Central (Backend Brain)
**Location**: `/central/` directory
**What it does**:
- Main API server that handles all business logic
- Stores all data in PostgreSQL database
- Enforces security policies and generates alerts
- Serves the REST/gRPC APIs that the UI consumes

**Tech Stack**: Go, PostgreSQL, gRPC

**For Frontend Engineers**: This is your backend. All API calls from the UI go to Central. It exposes:
- REST APIs at `/v1/`, `/v2/` endpoints
- GraphQL API at `/api/graphql`

### 2. UI (What You See)
**Location**: `/ui/` directory
**What it does**:
- Web-based dashboard for security insights
- Shows vulnerabilities, policy violations, and compliance data
- Allows configuration of policies and integrations

**Tech Stack**:
- React 18 with TypeScript
- PatternFly components (Red Hat's design system)
- Vite for bundling
- Apollo Client for GraphQL
- Axios for REST APIs

**For Frontend Engineers**: This is your home! The UI is a single-page React app that:
- Runs on port 3000 locally (proxies to Central on 8000)
- Talks to Central's APIs
- Uses modern React patterns (hooks, context, not Redux for new code)

### 3. Sensor (Cluster Monitor)
**Location**: `/sensor/` directory
**What it does**:
- Deployed in each monitored Kubernetes cluster
- Watches what's happening in the cluster (pods, deployments, etc.)
- Sends security events back to Central
- Acts as the "eyes and ears" in each cluster

**Tech Stack**: Go, Kubernetes client libraries

**For Frontend Engineers**: You rarely interact with Sensor directly. It feeds data to Central, which your UI displays.

### 4. Scanner (Vulnerability Scanner)
**Location**: `/scanner/` directory
**What it does**:
- Scans container images for known vulnerabilities (CVEs)
- Uses vulnerability databases to find security issues
- Returns scan results to Central

**Tech Stack**: Go, ClairCore

**For Frontend Engineers**: When you see CVE data in the UI (like "Image X has 5 critical vulnerabilities"), that data comes from Scanner.

### 5. roxctl (CLI Tool)
**Location**: `/roxctl/` directory
**What it does**:
- Command-line tool for administrators
- Used in CI/CD pipelines
- Handles tasks like image scanning, policy checks, backups

**Tech Stack**: Go, Cobra CLI framework

**For Frontend Engineers**: Think of this as the "terminal version" of the UI. Some users prefer CLI over GUI.

### 6. Operator (Kubernetes Operator)
**Location**: `/operator/` directory
**What it does**:
- Manages the lifecycle of StackRox components
- Handles installation, upgrades, and configuration
- Uses Kubernetes CRDs (Custom Resource Definitions)

**Tech Stack**: Go, Kubernetes Operator SDK

**For Frontend Engineers**: This is infrastructure code. You won't interact with it much.

## Data Flow (Frontend Perspective)

Here's how data flows when you view a page in the UI:

```
User opens UI
    ↓
UI (React) makes API call
    ↓
Central receives request
    ↓
Central queries PostgreSQL or processes data
    ↓
Central sends response (JSON)
    ↓
UI renders the data with PatternFly components
```

Example: Viewing the **Vulnerabilities page**
1. UI calls `/v1/images` API
2. Central queries PostgreSQL for image data
3. Central calls Scanner for vulnerability details
4. Central combines data and returns JSON
5. UI displays images with their CVE counts

## API Patterns

### REST APIs
- **Base path**: `/v1/` or `/v2/`
- **Example**: `GET /v1/alerts` returns list of security alerts
- **Usage**: Most CRUD operations (Create, Read, Update, Delete)

### GraphQL API
- **Base path**: `/api/graphql`
- **Usage**: Complex queries with nested data
- **Example**: Fetching deployment details with related policies in one request

**For Frontend Engineers**: The codebase is migrating towards GraphQL for complex queries, but REST is still heavily used.

## Local Development Setup (Quick Version)

For frontend-only development:

1. **Deploy backend locally** (one-time setup):
   ```bash
   ./deploy/k8s/deploy-local.sh
   ```

2. **Start UI dev server**:
   ```bash
   cd ui/apps/platform
   npm ci
   npm run start
   ```

3. **Access UI**: https://localhost:3000
   - Proxies API calls to Central at localhost:8000

## Key Directories for Frontend Engineers

| Directory | What's Inside |
|-----------|---------------|
| `/ui/` | All frontend code |
| `/ui/apps/platform/src/` | React components and pages |
| `/ui/apps/platform/src/services/` | API client functions |
| `/ui/apps/platform/cypress/` | E2E tests |
| `/proto/` | API definitions (generates TypeScript types) |
| `/central/graphql/` | GraphQL schema definitions |

## Common Frontend Tasks

### Adding a New Page
1. Create component in `/ui/apps/platform/src/Containers/YourPage/`
2. Add route in route configuration
3. Add API service call in `/ui/apps/platform/src/services/`
4. Use PatternFly components for UI
5. Add E2E test in `/ui/apps/platform/cypress/`

### Calling a Backend API
```typescript
// Example REST API call
import { fetchAlerts } from 'services/AlertsService';

const alerts = await fetchAlerts();
```

### Testing
```bash
# Unit tests
npm run test

# E2E tests
npm run test-e2e

# Linting
npm run lint
```

## Security & Communication

- **All communication uses mTLS** (mutual TLS) for security
- Central ↔ Sensor: gRPC over mTLS
- UI ↔ Central: HTTPS with JWT tokens
- Scanner ↔ Central: gRPC over mTLS

**For Frontend Engineers**: Authentication is handled via JWT tokens. The UI stores tokens and includes them in API requests.

## Deployment Models

### Local Development
- Everything runs on Docker Desktop or similar
- One cluster with all components

### Production
- **Central Services** (Central, Scanner, UI, DB) in management cluster
- **Secured Cluster Services** (Sensor, Admission Controller) in each monitored cluster
- Central can monitor 10s to 100s of clusters

## Tech Stack Summary (Frontend Focus)

| Component | Language | Framework |
|-----------|----------|-----------|
| UI | TypeScript | React 18, PatternFly |
| Central (Backend) | Go | gRPC, PostgreSQL |
| APIs | - | REST, GraphQL |

## Common Terminology

- **Central**: The backend/control plane
- **Sensor**: Agent in each cluster
- **Secured Cluster**: A cluster being monitored
- **Policy**: Security rule (e.g., "no privileged containers")
- **Violation**: When a policy is broken
- **CVE**: Common Vulnerabilities and Exposures (security bugs)
- **Admission Controller**: Blocks deployments that violate policies

## Resources for Frontend Engineers

- **UI README**: `/ui/README.md`
- **UI CLAUDE.md**: `/ui/CLAUDE.md` (detailed dev guide)
- **API Docs**: Generated from Central at `/docs/api`
- **GraphQL Schema**: `/central/graphql/README.md`

## Questions Frontend Engineers Often Ask

**Q: Where are the API endpoints defined?**
A: REST APIs are in `/central/*/service/service.go` files. GraphQL schema is in `/central/graphql/resolvers/`.

**Q: How do I mock API calls in tests?**
A: Use Vitest's mocking for unit tests. For E2E, use Cypress intercepts.

**Q: Why PatternFly instead of Material UI or Ant Design?**
A: StackRox is part of Red Hat's ecosystem, which standardizes on PatternFly.

**Q: Can I use Redux for state management?**
A: No, the codebase is actively migrating away from Redux. Use React Context and hooks for new code.

**Q: How do I see what API calls the UI makes?**
A: Open browser DevTools → Network tab. Filter by XHR/Fetch.

## Summary

StackRox is a Kubernetes security platform with a distributed architecture:
- **Central** is the backend brain (Go + PostgreSQL)
- **UI** is a React app (TypeScript + PatternFly)
- **Sensor** monitors each cluster and reports to Central
- **Scanner** finds vulnerabilities in container images
- **roxctl** is the CLI version of the UI

As a frontend engineer, you'll primarily work in `/ui/`, make API calls to Central, and display security data using PatternFly components. The backend does all the heavy lifting—your job is to make that data understandable and actionable for users.
