# On using LLMs

## TL;DR

The use of Large Language Models (LLMs) is treated as an implementation detail. Using an
LLM to assist your work is viewed no differently than using a search engine, consulting
Stack Overflow, or asking a friend for advice.

## A note for contributors

You are generally free to use any tools you choose, assuming they adhere to your legal and
corporate obligations. In any case, the expectations (e.g., code quality, testing,
documentation, etc) for contributions remain unchanged:

- understand entirely what you submit;
- ensure that what you submit works (solves the problem, compiles, passes tests);
- use your own judgement when addressing review comments.

As the author, *you bear responsibility* for every line of code, comment, and documentation
you submit.

You are not required to disclose LLM usage just as you are not required to disclose that
you used Google. However, if an LLM generated a significant or critical portion of your
contribution, we advise mentioning it—similar to how you would link to a StackOverflow
discussion to explain a counterintuitive snippet.

"The LLM wrote it" is not acceptable as a justification for bugs, security flaws,
technical debt, difficult to maintain code or any other shortcomings.

Be mindful of reviewer bandwidth. LLMs can significantly increase your output, but this
creates a bottleneck for those auditing your work. Increasing the frequency of PRs should
never be used to fatigue reviewers into accepting lower-quality contributions.

## A note for reviewers

Review the code, not the tool. If a submission appears to have hallucinated APIs,
incoherence, or subtle bugs typical of LLMs, reject the submission on the grounds of
correctness and quality, just as you would with poor human-written code.

Watch for verbosity. Pay attention to overcommunication typical of LLMs. Large volumes of
text—whether in code comments or PR descriptions—do not necessarily translate into better
information. Demand clarity and brevity just as you would with verbose human submissions.

## A note on the future

We recognize that software development methodologies are evolving. In the future, we may
adopt new paradigms—such as spec-driven development where implementation details are
entirely automated. Until some new process is formally adopted and documented, we continue
to operate under the standard model where the contributor is the primary agent of
understanding and quality.
