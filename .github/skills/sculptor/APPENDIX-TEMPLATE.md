# Appendix File Template

Create one appendix file per substantive research topic: `{idea-name}/appendix-{topic-name}.md`. These serve as deep-dive references that keep the main research document concise.

## When to Create an Appendix

- A research topic has enough detail to warrant its own document (competitor analysis, API exploration, technical deep-dive, benchmark results)
- External API responses or data samples are too large for inline inclusion
- A background research agent returns substantial findings
- The user shares reference material that needs summarization and commentary

## Structure

```markdown
# Appendix: {Topic Name}

## Summary
[2-3 sentence overview of what this appendix covers and why it matters to the idea]

## Findings

### {Subtopic A}
[Detailed findings, analysis, observations]

### {Subtopic B}
[Detailed findings, analysis, observations]

## Sample Data
[Representative payloads, API responses, schemas, or other raw data in fenced code blocks.
Include 2-3 realistic examples — enough to reveal edge cases like null fields, nested structures, or unexpected arrays.]

## Implications for Design
[How these findings constrain or inform the approach. What options does this open or close?]

## Sources
[Links, documents, or commands used to gather this information]
```

## Guidance

- **Link from research.md** — Every appendix should be referenced from the relevant section in the main research document.
- **Link from spec.md** — When the spec references external data formats, point to the appendix for full sample payloads rather than duplicating them inline.
- **Name descriptively** — `appendix-jira-api.md` not `appendix-1.md`.
- **Include raw data** — Sample API responses, CLI output, config file snippets. These are high-value for the implementing agent and are better preserved here than summarized away.
- **Surface shared design surfaces** — When research reveals data structures, config formats, or patterns that serve multiple interfaces (CLI, TUI, agent mode), call these out in the "Implications for Design" section.
