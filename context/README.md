# StackRox Backend Architecture for Frontend Engineers

This directory contains documentation explaining how StackRox's backend works, written specifically for frontend engineers. The goal is to help you understand the system architecture, data flows, and backend concepts so you can build better UI features.

## Philosophy

As a frontend engineer working on StackRox, you don't need to write Go code or understand every backend detail. But understanding **how** the backend works helps you:
- Build UI features that align with the backend's capabilities
- Debug issues more effectively
- Have better conversations with backend engineers
- Design UIs that match the system's actual behavior

## Getting Started

**New to StackRox?** Start here:
1. Read [Architecture Overview](./architecture-overview.md) - Understand how the distributed system works
2. Explore specific topics below based on what you're working on

## Documentation Index

### System Architecture

#### [Architecture Overview](./architecture-overview.md)
High-level system architecture: Central, Sensor, Scanner, and how they communicate. The hub-and-spoke model explained.

**When to read**: Day 1, or when you need to understand how the pieces fit together.

**What you'll learn**:
- What Central, Sensor, and Scanner do
- How data flows between components
- Why StackRox uses a hub-and-spoke architecture
- How your UI talks to the backend

### Backend Deep Dives

#### [01 - Central: APIs & Data Flow](./01-central-apis.md)
How Central serves API requests, stores data in PostgreSQL, and processes queries.

**When to read**: Before implementing any feature that calls backend APIs.

**What you'll learn**:
- How Central processes REST and GraphQL requests
- API request lifecycle (auth → authz → query → response)
- Why some queries are slow (joins, aggregations, scoping)
- Real-time vs cached data
- Pagination strategy
- Where data is stored (PostgreSQL schema)

#### [02 - RBAC: Backend Perspective](./02-rbac-backend.md)
How role-based access control is enforced in Central.

**When to read**: When debugging permission issues or building features with RBAC.

**What you'll learn**:
- Permission evaluation flow
- How RBAC filters database queries
- Resource hierarchy and scoping
- Why API calls return 403
- Performance impact of scoped access
- Permission caching (5-minute TTL)

#### Sensor: The Cluster Monitor _(Coming Soon)_
How Sensor monitors Kubernetes clusters and reports back to Central.

**Topics to cover**:
- What Sensor watches in a cluster
- How admission control works
- Real-time event streaming to Central
- Why some data appears delayed

#### Scanner: The Vulnerability Engine _(Coming Soon)_
How Scanner scans images for vulnerabilities (CVEs).

**Topics to cover**:
- Image scanning pipeline
- Vulnerability database updates
- Why scan results take time
- Delegated scanning

### Domain Concepts _(Coming Soon)_

#### Policies & Policy Enforcement
How security policies work end-to-end.

#### Alerts & Violations
The lifecycle of a security alert.

#### Deployments & Workloads
How StackRox understands Kubernetes workloads.

#### Images & Vulnerabilities
The vulnerability scanning and management pipeline.

#### Network Graph
How network traffic is monitored and visualized.

## Quick Reference

### Backend Component URLs (Local Development)

```bash
# Central API
https://localhost:8000/v1/...
https://localhost:8000/api/graphql

# UI Dev Server (proxies to Central)
https://localhost:3000
```

### Key Backend Directories

| Component | Location | What's Inside |
|-----------|----------|---------------|
| Central API | `/central/` | Go code for Central service |
| Sensor | `/sensor/` | Go code for cluster monitoring |
| Scanner | `/scanner/` | Go code for vulnerability scanning |
| API Definitions | `/proto/` | Protocol buffer definitions |
| Database Migrations | `/migrator/` | PostgreSQL schema migrations |

### Backend Tech Stack Summary

| Component | Language | Database | Communication |
|-----------|----------|----------|---------------|
| Central | Go | PostgreSQL | gRPC, REST, GraphQL |
| Sensor | Go | - | gRPC (to Central) |
| Scanner | Go | - | gRPC (to Central) |
| UI | TypeScript/React | - | REST/GraphQL (to Central) |

## How to Use This Documentation

1. **Start with architecture-overview.md** - Get the big picture first
2. **Dive into specific topics** - Read deep dives when you're working on related features
3. **Ask questions** - If something is unclear or missing, ask the team and update these docs
4. **Keep it updated** - As you learn how the backend works, document it here

## What This Documentation Is NOT

This is **not**:
- A guide to writing Go code (you don't need to)
- Comprehensive backend API documentation (use `/main/apidocs` for that)
- A replacement for UI development guides (see `/ui/CLAUDE.md` for UI patterns)

This **is**:
- An explanation of how the backend works from a frontend perspective
- Help for understanding why the system behaves the way it does
- Context for building UI features that match backend capabilities

## Additional Resources

### For Frontend Development
- **UI Development Guide**: `/ui/CLAUDE.md` - Patterns for React, PatternFly, testing, etc.
- **UI README**: `/ui/README.md` - Setup and build instructions

### For Backend Understanding
- **Main README**: `/README.md` - Full development setup
- **AGENTS.md**: `/AGENTS.md` - Architecture overview and common commands
- **API Documentation**: `https://localhost:8000/main/apidocs` (when Central is running)
- **Backend Code**: Explore `/central/`, `/sensor/`, `/scanner/` directories

### For Protobuf/API Types
- **Proto Definitions**: `/proto/` - API contract definitions
- **Generated Types**: `/ui/apps/platform/src/types/*.proto.ts` - TypeScript types from proto

---

**Last Updated**: 2025-10-09
**Maintained By**: Frontend Engineering Team (with help from Backend teammates!)
