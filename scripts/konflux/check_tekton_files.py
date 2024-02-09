#!/usr/bin/env python3

import os
import subprocess
import sys
import yaml

failures = []


def fail(filename, msg):
    failures.append(f"[{filename}] {msg}")


def check_top_level_keys(filename, data):
    expected_keys = set(["apiVersion", "kind", "metadata", "spec", "status", "taskRunTemplate"])

    extra_keys = data.keys() - expected_keys
    if extra_keys:
        fail(filename, "Extra top level key(s): %s" % ", ".join(extra_keys))

    missing_keys = expected_keys - data.keys()
    if missing_keys:
        fail(filename, "Missing top level key(s): %s" % ", ".join(missing_keys))


def check_spec_keys(filename, data):
    expected_keys = ["params", "workspaces", "pipelineSpec"]

    keys = data["spec"].keys()

    extra_keys = keys - set(expected_keys)
    if extra_keys:
        fail(filename, "Extra spec key(s): %s" % ", ".join(extra_keys))

    missing_keys = set(expected_keys) - keys
    if missing_keys:
        fail(filename, "Missing spec key(s): %s" % ", ".join(missing_keys))

    if extra_keys or missing_keys:
        # Order errors would be duplicative if we've already identified an error above
        return

    if list(keys) != expected_keys:
        fail(filename, f"Wrong order of spec keys: {list(keys)}; Expected: {expected_keys}")


def check_blank_lines(filename, data):
    config = {
        "blank_line_above": ["metadata", "spec"],
        "blank_lines_within": {
            "spec": {
                "blank_line_above": ["params", "workspaces", "pipelineSpec"],
                "blank_lines_within": {
                    "pipelineSpec": {
                        "blank_line_above": ["finally", "params", "results", "workspaces", "tasks"],
                        "blank_lines_within": {
                            "tasks": {}
                        },
                    },
                },
            },
        },
    }

    with open(os.path.join(".tekton", filename)) as fp:
        lines = list(l.rstrip() for l in fp.readlines())
        _check_blank_lines(filename, lines, config)


def _check_blank_lines(filename, lines, config, key="", prefix=""):
    prev_blank_line = False
    started = key == ""
    for i, line in enumerate(lines, 1):
        # Lines before the start of the pertinent section (prefix + key) should be ignored
        # prefix is synonymous with indentation
        if not started and f"{prefix[:-2]}{key}:" == line:
            started = True

        blank_line = len(line.strip()) == 0

        if not started or not (line.startswith(prefix) or line.startswith(f"{prefix[:-2]}-")):
            prev_blank_line = blank_line
            continue

        if not config:
            # empty config here implies this section is a list
            # there should be a blank line between each item in the list
            if line.startswith(f"{prefix[:-2]}- ") and not prev_blank_line:
                fail(filename, f"There should be a blank line above line {i}: {line}")
        else:
            # non-empty config here implies this section is a dict
            # there should be a blank line above every key in the dict that's in the `blank_line_above` list
            if any(line.startswith(f"{prefix}{k}:") for k in config["blank_line_above"]) and not prev_blank_line:
                fail(filename, f"There should be a blank line above line {i}: {line}")

        prev_blank_line = blank_line

    for name, section in config.get("blank_lines_within", {}).items():
        _check_blank_lines(filename, lines, section, name, prefix=prefix + "  ")


def check_list_indentation(filename, data):
    prev_space_prefix_len = 0
    with open(os.path.join(".tekton", filename)) as fp:
        for i, line in enumerate(fp.readlines(), 1):
            if len(line.strip()) == 0:
                continue

            space_prefix_len = len(line) - len(line.lstrip())
            if line.strip().startswith("-") and space_prefix_len > prev_space_prefix_len:
                fail(filename, f"Wrong list indendation at line {i}: expected {prev_space_prefix_len}, but got {space_prefix_len}: {line.rstrip()}")

            prev_space_prefix_len = space_prefix_len


def is_tekton_yaml(data):
    return data.get("apiVersion") == "tekton.dev/v1" and data.get("kind") == "PipelineRun"


def run_checks(fname, data):
    check_top_level_keys(fname, data)
    check_spec_keys(fname, data)
    check_list_indentation(fname, data)
    check_blank_lines(fname, data)


def check_tekton_file(file_path):
    with open(file_path, 'r') as file:
        data = yaml.safe_load(file)

    if is_tekton_yaml(data):
        run_checks(os.path.basename(file_path), data)
    else:
        failures.append("Non-Tekton YAML file in .tekton dir: %s" % yaml_file_path)


def check_tekton_files():
    for file in os.listdir(".tekton"):
        if file.endswith(".yaml") or file.endswith(".yml"):
            check_tekton_file(os.path.join(".tekton", file))
        else:
            failures.append("Non-YAML file in .tekton dir: %s" % file)

    if len(failures) > 0:
        print("Failures:")
        for failure in failures:
            print(f"* {failure}")
        sys.exit(1)
    else:
        print("Success!")


if __name__ == "__main__":
    check_tekton_files()
