# Contributing to moss

We look forward to your contributions, but ask that you first review
these guidelines.

### Sign the CLA

As moss is a Couchbase project we require contributors accept the
[Couchbase Contributor License
Agreement](http://review.couchbase.org/static/individual_agreement.html). To
sign this agreement log into the Couchbase [code review
tool](http://review.couchbase.org/).

### Submitting a change for review

All types of contributions are welcome, but please keep the following in mind:

- If you're planning a large change, you should really discuss it in a
  github issue or on the google group first.  This helps avoid
  duplicate effort and spending time on something that may not be
  merged.
- Existing tests should continue to pass, and new tests for the
  contribution are nice to have.
- All code should have gone through `go fmt`
- All code should pass `go vet`
- All code should pass the fuzz tests: please see README-smat.md.
