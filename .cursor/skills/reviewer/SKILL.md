---
name: reviewer
description: Conducts a comprehensive code review by exploring the codebase, selecting tech-stack-specific checklist modules, and producing a structured report with spec deviation findings and code quality findings in separate sections. Use when asked to review a codebase, audit code quality, check spec compliance, or assess production readiness.
---

# Reviewer

Explores a codebase, selects relevant checklist modules, and produces a structured review report. Spec deviations and code quality findings are always in separate report sections.

## Rules

1. **Explore first** — Complete all discovery steps before evaluating any checklist items. Never review code you haven't read.
2. **Read-only** — Do not modify any files in the target codebase. Only produce the report.
3. **Select, don't include all** — Include only checklist modules relevant to the detected tech stack. A Go project never gets TypeScript checks.
4. **Keep sections separate** — Spec deviations go in the Spec Deviations section. Code quality findings go in the Code Quality section. Never mix them.
5. **Be specific** — Every finding must include a real `file:line` reference. Every anti-pattern must name actual files discovered during exploration.

## Step 1: Discovery

Read at least 10-15 representative files before proceeding. Determine:

**Language & Framework**
- Primary language(s) — check `go.mod`, `package.json`, `pyproject.toml`, `Cargo.toml`, `pom.xml`, `build.gradle`
- Key frameworks (web, ORM, testing, async runtime)
- Build system

**Project Structure**
- All source directories and their purposes
- Entry points (`main`, `index`, `app`)
- Test files and test framework

**Spec / Requirements**
- Search for: `SPEC.md`, `PRD.md`, `docs/spec.md`, `docs/spec/` (directory — read all files within), `docs/plan.md`, `docs/design.md`, `docs/architecture.md`, `README.md`, `docs/adr/`
- Also check: GitHub issues referenced in code, RFCs, any `*.spec.*` or `*.requirements.*` files
- If none found: note "no formal spec — code-quality review only"

**Config & Deployment**
- Config files (YAML, JSON, TOML, env)
- Dockerfile, docker-compose, Helm, Terraform
- CI/CD (`.github/workflows`, `.gitlab-ci.yml`, `Jenkinsfile`, `.circleci`)
- Linter config (`.eslintrc`, `.golangci.yml`, `pyproject.toml [tool.ruff]`, etc.)

**Architecture**
- Style: monolith, microservices, event-driven, CLI, library, etc.
- Patterns: dependency injection, repository, CQRS, middleware, plugin system
- External integrations: APIs, databases, message queues, cloud services
- Dependency direction between packages/modules

**Project Conventions**
- Check `CLAUDE.md`, `CONTRIBUTING.md`, `.editorconfig`, code style guides
- Note project-specific naming conventions and architectural rules

## Step 2: Checklist Selection

Always include the Universal modules (U, T, P, A, D). Then add:

- **Language module** matching the detected primary language (Go, TypeScript, Python, Rust, Java/Kotlin)
- **Framework modules** for detected frameworks (Temporal, React/Frontend)
- **Infrastructure modules** if database or HTTP layers are detected (DB, API)

See [CHECKLIST-CATALOG.md](CHECKLIST-CATALOG.md) for all module IDs, check descriptions, and the Module Selection Matrix.

## Step 3: Spec Traceability

**If a spec was found:**
1. Read it completely
2. For each spec section or requirement, identify the implementation file(s) that fulfill it
3. Build a traceability matrix: `spec section → file:line`
4. Note any spec requirements with no identifiable implementation (missing features)

**If no spec was found:** skip this step, note it in the report, and redistribute the Spec Compliance scorecard weight.

## Step 4: Code Review

Read in bottom-up order (dependencies before dependents):
1. Domain models / data types
2. Business logic / services
3. Handlers / controllers / CLI commands
4. Tests

For each selected checklist item:
- Evaluate the codebase against the check
- If a violation is found: record severity, `file:line`, what was expected, what was found
- If check passes: note it as passing (for the appendix)

Also identify 5-15 project-specific anti-patterns based on the tech stack and patterns observed during exploration. For each: what to look for, where (specific files), and why it matters.

## Step 5: Report

Read [REPORT-TEMPLATE.md](REPORT-TEMPLATE.md) and produce the report following that structure exactly.

- **Spec Deviations section**: populate from the traceability matrix and spec checks. Omit this section entirely (do not leave it empty) if no spec was found.
- **Code Quality Findings section**: populate from checklist evaluation, grouped by module.
- **Scorecard**: adjust category weights based on project type — a library weights API design higher; a spec-driven project weights compliance higher; if no spec, redistribute that 25% across remaining categories.

Write the completed report to **`code-review.md`** in the project root, unless the user specifies a different output path.

## Anti-Patterns (DO NOT DO)

- **Skip Step 1** and jump straight to reading code — you will miss the spec, miscategorize the tech stack, and select wrong modules
- **Include all checklist modules** regardless of what was detected — Python checks on a Go project produce false findings
- **Mix spec deviations and code quality** findings in the same section — they have different ownership and remediation paths
- **Write generic anti-patterns** like "check for null pointer dereferences" without naming the actual files where the risk was observed
- **Guess `file:line` references** instead of reading the file — always verify line numbers before recording them
- **Leave report template placeholders unfilled** — every `{placeholder}` must be replaced with real content
- **Use the Spec Compliance scorecard weight when no spec exists** — redistribute the 25% to remaining categories
- **Record a finding without Expected/Actual** — both fields are required for actionable findings
- **Report Info-level findings as High or Critical** — use the severity definitions from CHECKLIST-CATALOG.md
- **Stop after finding the first issue per check** — read enough of each file to assess the pattern, not just the first instance

## Reference Files

| File | Purpose | Load during |
|------|---------|-------------|
| [CHECKLIST-CATALOG.md](CHECKLIST-CATALOG.md) | All checklist modules and selection matrix | Step 2 |
| [REPORT-TEMPLATE.md](REPORT-TEMPLATE.md) | Standard report structure | Step 5 |
