# Pull request guidelines

## Creating a PR

- **Title:** The title of your PR needs to be descriptive. It should be short enough to fit into the title box, and not bleed into the
  PR text body. If the PR addresses a JIRA ticket, it should be of the
  form `ROX-1234: Add feature X`. If the PR addresses multiple JIRA tickets, those should be comma-separated
  (`ROX-1234, ROX-5678: Fix a annoying things Y and Z`). If the PR is not related to a JIRA ticket, omit the JIRA indication,
  but still choose a descriptive title (`Work around provisioning failures on $platform`). *Simply using the JIRA ticket as the
  PR title is not sufficient*.
- **Description:** Describe the motivation for this change, or why some things were done a certain way. This section can be omitted
  if the change is straightforward (keep in mind that the title has to describe what the change does). If you think a description
  is not necessary, remove the entire section; otherwise, remove the placeholder text only and fill in your own description.
- **Checklist:** The boxes should be self-explanatory. Only check them after having completed the respective task, and pushed all
  relevant changes. In many cases, some of the tasks do not apply to your PR. In this case, strike out the text next to the checkbox by
  enclosing it in `~...~`, and mark the box as checked (you can do so at PR creation time by replacing the `[ ]` with `[x]`). The
  `If any of these don't apply, ...` placeholder text can be removed, but that is not a requirement.
- **Testing performed:** Typical entries here include `CI`, `added a unit test for X`. For complex, functional features, you can also
  include detailed manual testing/verification instructions ([example](https://github.com/stackrox/rox/pull/3978)). It is recommended to
  populate this section early on to include the testing steps you _intend_ to do, and annotate them with `(TBD)`, or add a task checkbox via `[ ]`,
  to indicate they haven't been done yet. Make sure all `(TBD)`s are removed/all tasks are completed before checking the box and merging the PR.
- **PR type:** If you are creating a PR primarily to give CI a first shot, and don't intend anyone to review the PR yet, create a [*Draft* PR](https://github.blog/2019-02-14-introducing-draft-pull-requests/)
  by clicking on the small downward arrow next to the "Create pull request" button, and selecting "Create draft pull request" (if you accidentally
  created a non-draft PR, you can convert it back to draft, see below for instructions).

## Working on a PR

### Git operations
- Create a separate Git commit for every set of incremental changes.
- Do not use `git commit --amend` on changes you have already pushed to the remote.
- Only use `git rebase` (or the `smart-rebase` script) for rebasing on top of latest `master` changes. Do not use merge commits.
- Never locally squash changes, except for making conflict resolution during a `git rebase` easier. Always try rebasing
  without squashing first; if there are conflicts that you believe will be easier to resolve with squashing, do `git rebase --abort`
  followed by squashing and re-running the rebase command.
- The only case in which you need to force-push to your branch is after rebasing on top of latest master changes.

### Interaction with reviewers
- Consult the relevant style guide ([golang](go-coding-style.md)) before creating a PR.  
- It is preferable to respond to reviewer comments in the "Files changed" view. In contrast to the "Conversation" view, this allows you
  to store your responses as drafts and submit them in a single batch. It is advisable to only submit your responses in GitHub after
  pushing the changes addressing the reviewers's comments, this avoids confusion.
- Do not unilaterally resolve comment threads if you merely believe to have addressed a comment. Usually, it should be left to the reviewer to
  resolve their comments. This is not a hard and fast rule; there might be trivial cases, or cases where a reviewer replies with a comment
  like "OK, makes sense", indicating agreement, without resolving the comment thread. If you feel seriously inhibited by expanded comment
  threads in the "changed" view, resolve them; when there is any doubt about the reviewer being OK with the resolution, leave them
  unresolved.
- After addressing a set of review comments, indicate that the PR is ready for a new round of reviews, click on the small "cycle/refresh"
  icon next to their name (![icon](images/re-request-review.png?raw=true)).
- If a PR is time-sensitive, or has been dragging on for a long time, you should absolutely feel comfortable pinging the reviewer(s)
  via Slack to expedite the review process. There is no minimum time you have to wait before doing so. Often, reviewers are flexible enough to preempt
  current activities for urgent reviews, but they cannot be expected to proactively do so with low latency, as that would require
  constantly monitoring the PR overview or their inbox to see if new
  PRs need action. However, use good judgment, and don't interrupt people for changes that are of low relevance. If the reviewer tells you
  that they currently do not have the bandwidth to review within the next couple of hours or even days, do not hesitate to look for
  other reviewers (possibly via Slack).
- Usually, the "changes requested" status by a reviewer should not be dismissed. An exception is the case when the
  reviewer is OOO, in a different time zone, or is not able to do another round of reviews in the near future,
  and the PR is time sensitive (or has been dragging on for an unreasonably long time). However, you should always
  check with the reviewer first via a quick Slack message (regardless of OOO status, local timezone etc.). If they do
  not reply after a couple of minutes, you can dismiss their review, recording via a pull request comment on GitHub why
  you chose to do so.


### Metadata
- Use labels like `blocked`, `on hold`, `needs reviewer` etc. to communicate the state of your PR.
- If you mistakenly believed a PR was ready for reviewer and turned it from a draft PR to a regular PR, you can revert
  it to a draft PR by clicking "Convert to draft" under the list of reviewers in the right sidebar of the Conversation view.

## Merging a PR

- Make sure that all CI statuses are green. If any CI steps fail, leave a comment on the PR explaining why you believe the failure
  is not caused by your PR (e.g., `Test X is failing on master` or `Test Y has been flaky`, or `Provisioning/UI/backend failures unrelated
  as they are not affected by changes in this PR`).
- Ensure that every item in the checklist is either checked or crossed out. Also make sure there are no `TBD` entries under
  "Testing performed".
- Always use `Squash and merge` as the merging mode (default).
- Double-check that the title of the commit ("subject line") is your PR title, followed by the PR number prefixed with a `#` in
  parentheses (e.g., `ROX-1234: Add feature X (#5678)`). In some cases GitHub might pre-fill something different here, especially if the
  title of the PR has changed over the course of its existence.
- The body of the commit message should normally be empty. GitHub pre-populates it with the subject lines of all individual commits; delete this.
  It needlessly inflates the Git log; the canonical location for any additional details is the PR page (locatable through the PR
  number referenced in the subject line).
  If you think that this will help future readers of the code that inspect commit logs (e.g., via `git blame`), you can
  add some additional context in the commit message body. Consider however that this has low visibility, expect most people
  to never read it.
- After merging a PR, keep an eye out on the [`#test-failures`](https://srox.slack.com/archives/CLUNQEEMA) Slack channel in case
  your merged PR caused any breakages on `master`.
