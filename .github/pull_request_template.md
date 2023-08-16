## Description

A detailed explanation of the changes in your PR.

Feel free to remove this section if it is overkill for your PR, and the title of your PR is sufficiently descriptive.

## Checklist
- [ ] Investigated and inspected CI test results
- [ ] Unit test and regression tests added
- [ ] Evaluated and added CHANGELOG entry if required
- [ ] Determined and documented upgrade steps
- [ ] Documented user facing changes (create PR based on [openshift/openshift-docs](https://github.com/openshift/openshift-docs) and merge into [rhacs-docs](https://github.com/openshift/openshift-docs/tree/rhacs-docs))

If any of these don't apply, please comment below.

## Testing Performed

TODO(replace-me)  
Use this space to explain **how you validated** that **your change functions exactly how you expect it**.
Feel free to attach JSON snippets, curl commands, screenshots, etc. Apply a simple benchmark: would the information you
provided convince any reviewer or any external reader that you did enough to validate your change.

It is acceptable to assume trust and keep this section light, e.g. as a bullet-point list.

It is acceptable to skip testing in cases when CI is sufficient, or it's a markdown or code comment change only.  
It is also acceptable to skip testing for changes that are too taxing to test before merging. In such case you are
responsible for the change after it gets merged which includes reverting, fixing, etc. Make sure you validate the change
ASAP after it gets merged or explain in PR when the validation will be performed.  
Explain here why you skipped testing in case you did so.

### Reminder for reviewers

In addition to reviewing code here, reviewers **must** also review testing and request further testing in case the
performed one does not seem sufficient. As a reviewer, you must not approve the change until you understand the
performed testing and you are satisfied with it.
