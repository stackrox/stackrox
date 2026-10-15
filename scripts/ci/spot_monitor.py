#!/usr/bin/env python3
"""GitHub Actions adapter for the shared nongroovy node monitor."""

import argparse
import os
from pathlib import Path
import subprocess
import sys

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / '.openshift-ci'))
import node_disruption  # noqa: E402  # pylint: disable=wrong-import-position


def run_link():
    run_id = os.getenv('GITHUB_RUN_ID')
    if not run_id:
        return ''
    server = os.getenv('GITHUB_SERVER_URL', 'https://github.com').rstrip('/')
    repository = os.getenv('GITHUB_REPOSITORY', '')
    attempt = os.getenv('GITHUB_RUN_ATTEMPT', '1')
    return f'{server}/{repository}/actions/runs/{run_id}/attempts/{attempt}'


def publish_output(failed):
    output = os.getenv('GITHUB_OUTPUT')
    if not output:
        return
    with open(output, 'a', encoding='utf-8') as stream:
        stream.write(f"failed={str(failed).lower()}\n")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['run'])
    parser.add_argument('command', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ['--'] else args.command
    if not command:
        parser.error('a command is required after --')
    if not node_disruption.enabled():
        return subprocess.call(command)
    try:
        result = node_disruption.run(command, run_url=run_link())
    except (OSError, RuntimeError) as exc:
        publish_output(True)
        print(f'::error::Node monitor could not finish: {exc}', flush=True)
        return 1
    publish_output(result.monitor_failed)
    if result.monitor_failed:
        print('::error::Node disruption or monitor error; see node-monitor.xml', flush=True)
    return result.exit_code


if __name__ == '__main__':
    raise SystemExit(main())
