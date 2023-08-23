# 0000 - Adopt ADRs

- **Author(s):** J. Victor Martins <jvdm@sdf.org>
- **Created:** [2023-08-23 Wed]

## Status

Accepted.

## Context

Currently, decisions about Stackrox Scanner are made using informal slack threads, ad-hoc discussions, and occasionally when the problem is considered complex, design documents. They are:

- **Hard to Navigate:** Finding discussions, decisions, and their context can sometimes be challenging. There is no decision log providing context on the implementation and design information.
- **Opportunistic:** Decisions might be made too fast without all relevant consideration.
- **Overlooks Long-Term Impact:** Quick decisions might only partially evaluate the long-term effects on the project.
- **Lacks a Well-Defined Lifecycle:** There's no structured review process to guarantee why a decision was accepted or rejected at the end.
- **Unclear Boundaries:** There's no clear framework to help us understand when the overhead of design documents is worth it.

## Decision

1. **ADRs for long-term impact decisions:** ADRs will be used to document decisions that have a long-term impact on the project. This process is considered more mature and reliable than other methods (e.g., slack threads).  We will continue to use informal methods for trivial decisions.
2. **ADRs for smaller and larger decisions:** ADRs can be used for smaller and larger decisions, whether public or not, depending on the context.
3. **Design docs to aid complex contexts:** Design documents can be used when the context is ambiguous and requires an in-depth analysis of options available. ADRs will still be used to formally record the decision.[^1]
4. **ADRs are public and local by default:** ADRs will be kept in a specific directory within the code repo to correlate decisions with code changes and make them widely available, even to external parties. The ADR will be stored in a private location when the context is sensitive (e.g., security, customer information).
5. **ADR template:** ADRs will use a well-defined template.

## Consequences

- **Additional overhead:** ADRs are now required when a few slack threads would be OK. That creates some overhead to long-term discussions and decisions that did not exist before.
- **Improved Clarity and Velocity:** This approach will bring clear communication, faster onboarding time, a better context for discussions, and reduce unnecessary back-and-forths in other contexts (e.g., PR).
- **Transparency and Accessibility:** By making ADRs accessible, internal team members and external stakeholders will benefit from increased visibility into the decision-making process.
- **Flexibility between ADRs and Design Docs:** A better understanding of when a design document will add value, providing flexibility to adapt to the evaluated problem.
- **Public vs. Private:** Private ADRs need to exist when the decision is sensitive to happen in public. Even though the number of private decisions will be significantly lower, having two ADR locations will still be painful.

[^1]: Design Documents can clarify decisions when the problem or context is too complex or ambiguous. Its purpose is to lay out options, compare them, find pros and cons, obtain data, and provide a recommendation based on the problem. Not all decisions require the same level of detail or overhead; we can also determine whether we need that by reviewing the ADR. Yes, the additional information could be added directly to the ADR, but leaving the Design Document format open has advantages. It can be modified and adapted to suit the problem it is evaluating. Also, PR-like reviews are not ideal for editing and discussing design docs. While ADRs are records that should not change with time (only if another ADR supersedes it etc.).

