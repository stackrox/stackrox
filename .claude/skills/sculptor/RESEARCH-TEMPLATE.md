# Research Template

Write findings to `{idea-name}/research.md` with these sections:

```markdown
# Research: {Idea Name}

## Problem Space
[What problem exists, who has it, why it matters]

## Prior Art
[Existing solutions, competitors, relevant projects]

## Technical Landscape
[Relevant technologies, constraints, opportunities]

## Key Insights
[What we learned that shapes the approach]

## Out of scope
[Things we have explicitly decided to remove from scope]

## Open Questions
[Things we still need to figure out]
```

## Appendix Files

For any topic that warrants a deep dive (competitor analysis, API exploration, benchmark results, technical deep-dives), create a separate appendix file rather than bloating the research document. See [APPENDIX-TEMPLATE.md](APPENDIX-TEMPLATE.md) for the format.

Link each appendix from the relevant section above, e.g.:
```markdown
## Prior Art
...detailed analysis in [appendix-competitor-analysis.md](appendix-competitor-analysis.md)

## Technical Landscape
...API response formats documented in [appendix-stripe-api.md](appendix-stripe-api.md)
```
