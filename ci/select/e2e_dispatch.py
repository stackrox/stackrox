import json
import os
import subprocess
from types import SimpleNamespace
from dataclasses import dataclass

@dataclass
class Job:
    name: str
    require_pr_label: bool = False
    require_konflux_build: bool = False
    is_ui_test: bool = False

    def should_run(self, input):
        if input.is_fork == 'true':
            return False

        if input.event_name != 'pull_request':
            return True
        elif self.name in input.pr_labels:
            return True

        # We are in a PR context, job is not explicitly listed among the PR labels.

        # TODO(ROX-21073): re-enable Konflux for all PR branches
        if self.require_konflux_build and 'konflux-build' not in input.pr_labels:
            return False

        if input.ui_only_change == 'true':
            # UI test jobs should run on UI-only changes; all other jobs should skip.
            return self.is_ui_test
        if self.require_pr_label:
            return False
        if input.is_draft != 'true':
            return True

        return False


input = SimpleNamespace(**{
    key.removeprefix('input__'): val
    for (key, val) in os.environ.items()
    if key.startswith('input__')})

if input.event_name != 'pull_request':
    input.pr_labels = []
else:
    p = subprocess.run(
        ['gh', 'pr', 'view', input.pr_number, '--repo', input.repo, '--json', 'labels', '--jq', '.labels[].name'],
        check=True, encoding='utf-8', stdout=subprocess.PIPE, stdin=subprocess.DEVNULL)
    input.pr_labels = p.stdout.strip().splitlines()

for k, v in vars(input).items():
    print(f'{k} = {v}')

jobs_registry = [
    Job(name='e2e-byodb-tests', require_pr_label=True),
    Job(name='e2e-db-backup-restore-tests-gke'),
    Job(name='e2e-db-backup-restore-tests-gke-konflux', require_konflux_build=True),
    Job(name='e2e-gke-upgrade-tests'),
    Job(name='e2e-nongroovy-tests-gke'),
    Job(name='e2e-nongroovy-tests-gke-konflux', require_konflux_build=True),
    Job(name='e2e-qa-tests-gke', require_pr_label=True),
    Job(name='e2e-qa-tests-gke-konflux', require_konflux_build=True),
    Job(name='e2e-qa-tests-ocp-fips-konflux', require_konflux_build=True),
    Job(name='e2e-ui-tests-gke', require_pr_label=True, is_ui_test=True),
    Job(name='e2e-ui-tests-gke-konflux', require_konflux_build=True, is_ui_test=True),
]

# Only valid, applied decisions from a successful PR planner can skip work.
planned = {}
if input.event_name == 'pull_request' and input.is_fork != 'true' and input.ci_planner_result == 'success':
    try:
        candidate = json.loads(input.ci_decisions or '{}')
        if not isinstance(candidate, dict) or any(value not in ('run', 'skip') for value in candidate.values()):
            raise ValueError('invalid selection map')
        planned = candidate
    except (ValueError, TypeError):
        print('CI selection unavailable; preserving existing e2e eligibility')
ci_actions = {job.name: planned.get(job.name, 'run') for job in jobs_registry}

with open(os.environ['GITHUB_OUTPUT'], mode='at', encoding='utf-8') as file:
    def write(job, val):
        line = f'{job}={val}'
        print(line)
        file.write(line + '\n')

    # Skipped suites still start their jobs so required checks can finish successfully.
    jobs = {job.name: job.should_run(input) or ci_actions[job.name] == 'skip' for job in jobs_registry}
    write('ci-actions', json.dumps(ci_actions))
    write('skip-gha-images', str(all(ci_actions[job.name] == 'skip' for job in jobs_registry if not job.require_konflux_build)).lower())
    write('skip-konflux-images', str(all(ci_actions[job.name] == 'skip' for job in jobs_registry if job.require_konflux_build)).lower())

    write('jobs', json.dumps(jobs))

    # Find out when we can complete the workflow early by not even waiting for GHA-built images.
    wait_for_gha_images = input.event_name != 'pull_request' or input.is_draft != 'true' or any(jobs.values())
    write('wait-for-gha-images', str(wait_for_gha_images).lower())

    # Waiting for konflux-built images in PR request context requires the 'konflux-build' PR label.
    wait_for_konflux_images = input.event_name != 'pull_request' or ('konflux-build' in input.pr_labels and any(jobs.values()))
    write('wait-for-konflux-images', str(wait_for_konflux_images).lower())
