#!/usr/bin/env python3

import datetime
import json
import os
import re


def parse_image_refs(image_refs):
    return json.loads(image_refs)


def validate_component(component):
    assert (
        component["name"] != ""
        and component["containerImage"] != ""
        and component["revision"] != ""
        and component["repository"] != ""
    ), "Component must have component name, ref, revision and repository set. Check container image labels."


def determine_component_version_suffix(application):
    match = re.search(r"acs-(?P<version>\d+-\d+)", application)
    if match:
        return match.group('version')
    return ""


def process_component(component, name_suffix):
    validate_component(component)
    if name_suffix != "":
        name = f"{component["name"]}{name_suffix}"
    else:
        name = component["name"]

    return {
        "containerImage": component["containerImage"],
        "name": name,
        "source": {
            "git": {
                "revision": component["revision"],
                "url": component["repository"]
            }
        }
    }


def construct_snapshot(
    snapshot_name_prefix,
    snapshot_version_suffix,
    pipeline_run_name,
    namespace,
    application,
    components
):
    timestamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    snapshot_name = f"{snapshot_name_prefix}-{snapshot_version_suffix}-{timestamp}".lower()
    return {
        "apiVersion": "appstudio.redhat.com/v1alpha1",
        "kind": "Snapshot",
        "metadata": {
            "name": snapshot_name,
            "namespace": namespace,
            "labels": {
                "appstudio.openshift.io/build-pipelinerun": pipeline_run_name
            }
        },
        "spec": {
            "application": application,
            "components": components
        }
    }


def write_snapshot(snapshot, results_path):
    with open("snapshot.json", "w") as f:
        json.dump(snapshot, f)
    with open(results_path, "w", newline="") as f:
        f.write(snapshot["metadata"]["name"])


if __name__ == '__main__':
    application = os.environ["APPLICATION"] # 1
    name_suffix = determine_component_version_suffix(application)
    image_refs = parse_image_refs(os.environ["IMAGE_REFS"]) # 2
    components = [process_component(c, name_suffix) for c in image_refs]

    main_image_tag = os.environ["MAIN_IMAGE_TAG"] # 3
    pipeline_run_name = os.environ["PIPELINE_RUN_NAME"] # 4
    namespace = os.environ["NAMESPACE"] # 5

    snapshot = construct_snapshot(
        snapshot_name_prefix=application,
        snapshot_version_suffix=main_image_tag,
        pipeline_run_name=pipeline_run_name,
        namespace=namespace,
        application=application,
        components=components
    )

    snapshot_name_results_path = os.environ["SNAPSHOT_NAME_RESULTS_PATH"]
    write_snapshot(snapshot, snapshot_name_results_path)
