# Shepherding code changes

In stackrox/stackrox we rely on "shepherds" to help advance the PRs, help with the
direction for a change, and ultimately help folks to work on what's important.

## What exactly is shepherding?

Shepherding is a generalization of the PR reviewer concept that goes beyond code reviews.
It is a well-established practice, both here and in the industry, to review code changes
before merging them, yet we do not often apply the same approach for other phases of the
feature development cycle. Similarly to a PR reviewer, a shepherd usually understands the
affected product area well, knows undocumented and non-obvious facts and effects, and has
a vision of how the product area will develop in the next year.

A shepherd helps you throughout the entire process of making a contribution, from the
design to analyzing bug reports. While you’re still responsible for the change overall,
the shepherd will make time to help you advance.

Overall, a shepherd is your partner in landing the change, someone with experience in the
affected area. Find shepherds for non-trivial code changes *before* making them and agree
with your shepherd on the approach and the timeline: shepherds commit to making space in
their schedule and should be able to help with maintaining the change.

## How shepherding works

To give an idea how shepherding works, here are some aspects of shepherding. These are not
the hard rules as situations are different and there are always exceptions. Some
components have a higher contribution barrier than others. Sometimes what looked trivial
becomes non-trivial and a shepherd is sought later in the development process.

- Shepherding is not a burden but your help in getting the stuff you need done. It reduces
  the chances of bugs and code rewrites; and helps avoid stress for getting last-minute
  reviews for your changes.
- Reach out to folks who you think are good shepherds for the work you plan. Ask around if
  you don’t know who can be a good candidate.
- If you don’t have a shepherd and no-one reviews your ~1K long PR days before the code
  freeze — tough luck. Next time find a shepherd earlier.
- A person you reach out to can say “no”. In such a case, ask for reasons and then ask
  someone else.
- If you struggle to find a shepherd, talk to your manager, or to another Red Hatter. This
  is healthy and is a natural way to focus the collective mind on what’s truly important.
  Don’t take it personally, and avoid working without a shepherd; try to figure
  out why this happened. Maybe we don’t focus on the right things as a project or maybe
  the effort you’re working on is not the top priority.
- For cross-component efforts, you will likely have a shepherd per component, and this is
  OK.
- Expect your shepherd to help with resolving conflicting views on PRs. They are the
  component maintainer, so they have the final say.
- Expect your shepherd to allocate time to answer your questions, review your design and
  code.
- A shepherd is not responsible for time and project management but for giving timely
  feedback.
