# Idea Template

Write to `{idea-name}/idea.md`:

```markdown
# {Idea Name}

## Problem
[Clear statement of the problem being solved]

## Context
[Background, constraints, assumptions]

## Proposed Approaches

### Approach A: {Name}
[Description, how it works, trade-offs]

### Approach B: {Name}
[Description, how it works, trade-offs]

### Approach C: {Name} (if warranted)
[Description, how it works, trade-offs]

## Recommendation
[Which approach and why]

## Open Questions
[Remaining uncertainties]
```

## Scaling

For simple ideas, collapse to Problem + Solution + Rationale. For complex ones, add sections as needed (data model sketches, API shapes, user flows, etc.).

## Guidance

- **Deferred features**: When specifying Phase 2+ features in detail, explicitly call out cross-feature dependencies and shared interfaces. Users often ask for details on deferred features "so we can see how they influence each other" -- surface these connections proactively.
- **Present design in sections** — Walk the user through each major section and get their reaction before moving on.
- **Reference appendix files**: When an approach relies on findings from a research appendix (API format, competitor behavior, technical constraint), link to it rather than restating. This keeps the idea document focused on the "what" while appendices hold the "evidence."
