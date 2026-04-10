# Markdown File Format for `bd create -f`

Write a `.md` file using this structure:

- `## Title` (H2) starts each issue
- `### Section` (H3) sets metadata within an issue
- Lines after `## Title` before any `###` become the description

## Recognized Sections

| Section | Content | Default |
|---------|---------|---------|
| `### Priority` | `0`-`4` or `P0`-`P4` | `2` |
| `### Type` | `bug`, `feature`, `task`, `epic`, `chore` | `task` |
| `### Description` | Multi-line text (overrides auto-description) | -- |
| `### Design` | Implementation approach, architecture notes | -- |
| `### Acceptance Criteria` | Definition of done, success criteria | -- |
| `### Assignee` | Username | -- |
| `### Labels` | Comma or space-separated | -- |
| `### Dependencies` | `blocks:id, discovered-from:id, parent-child:id` | -- |

## Example Plan File

```markdown
## Goal: Build Authentication System

### Type
epic

### Priority
0

### Description
End-to-end auth system with JWT tokens, login/logout, and password reset.

### Acceptance Criteria
- Users can register, login, and logout
- JWT tokens with refresh rotation
- Password reset via email

## Create User model with email, password_hash, created_at fields

### Type
task

### Priority
2

### Description
Create the User model in models/user.py with all required fields and migrations.

## Add POST /api/auth/login endpoint

### Type
task

### Priority
2

### Description
Login endpoint in routes/auth.py. Validates credentials, returns JWT access + refresh tokens.

## Add POST /api/auth/logout endpoint

### Type
task

### Priority
2

### Description
Logout endpoint that invalidates the refresh token.

## Write unit tests for authentication

### Type
task

### Priority
2

### Description
Tests for register, login, logout, and token refresh in tests/test_auth.py.
```

## Post-Creation Dependencies

Issues created in the same file cannot reference each other's IDs (unknown at creation time). Add cross-issue dependencies after creation:

```bash
bd create -f plan.md --json
# Parse returned IDs, then chain dependency additions using --blocks (blocker first, clear direction):
bd dep <user-model-id> --blocks <login-id> && bd dep <login-id> --blocks <logout-id> && bd dep <logout-id> --blocks <tests-id>
# Add hierarchy (dep add with parent-child type: child first, parent second):
bd dep add <user-model-id> <epic-id> -t parent-child && bd dep add <login-id> <epic-id> -t parent-child
```
