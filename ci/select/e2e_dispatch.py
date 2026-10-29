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

    def should_run(self, inputs):
        """Return whether the job is eligible for the workflow inputs.

        An explicit job label bypasses draft, UI-only, and build-label checks,
        but forked pull requests always skip the job.
        """
        if inputs.is_fork == 'true':
            return False

        if inputs.event_name != 'pull_request':
            return True
        elif self.name in inputs.pr_labels:
            return True

        # We are in a PR context, job is not explicitly listed among the PR labels.

        # TODO(ROX-21073): re-enable Konflux for all PR branches
        if self.require_konflux_build and 'konflux-build' not in inputs.pr_labels:
            return False

        if inputs.ui_only_change == 'true':
            # UI test jobs should run on UI-only changes; all other jobs should skip.
            return self.is_ui_test
        if self.require_pr_label:
            return False
        if inputs.is_draft != 'true':
            return True

        return False


def main():
    """Select E2E jobs and append their workflow outputs."""
    inputs = SimpleNamespace(**{
        key.removeprefix('input__'): val
        for (key, val) in os.environ.items()
        if key.startswith('input__')})

    if inputs.event_name != 'pull_request':
        inputs.pr_labels = []
    else:
        p = subprocess.run(
            ['gh', 'pr', 'view', inputs.pr_number, '--repo', inputs.repo, '--json', 'labels', '--jq', '.labels[].name'],
            check=True, encoding='utf-8', stdout=subprocess.PIPE, stdin=subprocess.DEVNULL)
        inputs.pr_labels = p.stdout.strip().splitlines()

    for k, v in vars(inputs).items():
        print(f'{k} = {v}')

    jobs_registry = [
        Job(name='e2e-byodb-tests', require_pr_label=True),
        Job(name='e2e-db-backup-restore-tests-gke'),
        Job(name='e2e-db-backup-restore-tests-gke-konflux', require_konflux_build=True),
        Job(name='e2e-gke-upgrade-tests'),
        Job(name='e2e-nongroovy-tests-gke'),
        Job(name='e2e-nongroovy-tests-gke-konflux', require_konflux_build=True),
        Job(name='e2e-operator-tests-gke'),
        Job(name='e2e-operator-tests-gke-konflux', require_konflux_build=True),
        Job(name='e2e-qa-tests-gke', require_pr_label=True),
        Job(name='e2e-qa-tests-gke-konflux', require_konflux_build=True),
        Job(name='e2e-qa-tests-ocp-fips-konflux', require_konflux_build=True),
        Job(name='e2e-ui-tests-gke', require_pr_label=True, is_ui_test=True),
        Job(name='e2e-ui-tests-gke-konflux', require_konflux_build=True, is_ui_test=True),
    ]

    with open(os.environ['GITHUB_OUTPUT'], mode='at', encoding='utf-8') as file:
        def write(job, val):
            line = f'{job}={val}'
            print(line)
            file.write(line + '\n')

        jobs = {job.name: job.should_run(inputs) for job in jobs_registry}

        write('jobs', json.dumps(jobs))

        # Find out when we can complete the workflow early by not even waiting for GHA-built images.
        wait_for_gha_images = inputs.event_name != 'pull_request' or inputs.is_draft != 'true' or any(jobs.values())
        write('wait-for-gha-images', str(wait_for_gha_images).lower())

        # Waiting for konflux-built images in PR request context requires the 'konflux-build' PR label.
        wait_for_konflux_images = inputs.event_name != 'pull_request' or ('konflux-build' in inputs.pr_labels and any(jobs.values()))
        write('wait-for-konflux-images', str(wait_for_konflux_images).lower())

        # Print a human-readable summary of which jobs will be dispatched.
        print()
        width = max(len(name) for name in jobs)
        for name in sorted(jobs):
            print(f'{name:<{width}}  {"✓" if jobs[name] else ""}')


if __name__ == '__main__':
    main()
