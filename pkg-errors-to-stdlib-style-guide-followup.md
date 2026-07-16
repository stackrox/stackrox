# Follow-up: update error-handling style guide from `pkg/errors` to stdlib `errors`

## Context

PR #21435 review comment M3 flagged that the new roxagent files (`serve.go`,
`mapping_refresher.go`, `tls.go`, plus their tests) use stdlib `errors` /
`fmt.Errorf("...: %w", err)` instead of `github.com/pkg/errors`'s
`errors.Wrap[f]()`, which is what `.github/go-coding-style.md` (lines 196-201)
currently documents as the convention:

```
- Use `errors.Wrap[f]()` from `github.com/pkg/errors` to add a message when
  forwarding the error.
- Use `RoxError.CausedBy[f]()` from `pkg/errox` to add context to an existing
  message.
- Prefer `RoxError.New[f]()` from `pkg/errox` over `errors.Errorf()` from
  `github.com/pkg/errors` and `errors.New()` from the _builtin_ errors package
  to assign the error one of the standard classes.
```

## Decision for PR #21435

Kept the new files on stdlib `errors`/`fmt.Errorf`, did **not** convert them
to `pkg/errors`. Rationale:

- Direction going forward is stdlib-first: native `%w` wrapping plus
  `errors.Is`/`As`/`Join`/`AsType` (the last one, used in `protocol.go`, has
  no `pkg/errors` equivalent at all) make `pkg/errors` largely redundant for
  its original purpose.
- Recently-added files repo-wide already skew stdlib over `pkg/errors`
  (~3:1 in a quick 3-month sample), so this isn't a new direction, just one
  the style guide hasn't caught up to.
- No lint rule depends on either style: `wrapcheck` is disabled for the
  entire `compliance/` tree in `.golangci.yml`, so nothing in CI enforces
  `pkg/errors` here.

## What the follow-up needs to do

1. Update `.github/go-coding-style.md`'s "Error handling" section to
   recommend stdlib `errors` + `fmt.Errorf("...: %w", err)` over
   `github.com/pkg/errors`'s `Wrap`/`Wrapf`. Leave the `pkg/errox`
   (`RoxError.New[f]()` / `CausedBy[f]()`) guidance as-is — that's a
   different, still-valid layer for classified errors, orthogonal to this
   decision.
2. Decide (separately, not part of the doc update itself) whether this is
   just "stop using `pkg/errors` in new code" or something that also
   prompts opportunistic migration of existing call sites. There are 1200+
   files on `pkg/errors` repo-wide — a blanket migration is its own,
   much larger effort and almost certainly not worth doing in one PR.
3. Optional, smaller follow-up: `compliance/node/index/indexer.go` was the
   "adjacent" file the M3 review comment cited as already using
   `pkg/errors`. Worth a standalone PR if/when someone touches that file
   anyway, not urgent on its own.

Not time-sensitive — this is a documentation-only change with no CI
dependency either way.
