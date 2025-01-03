#!/usr/bin/env python3

import datetime
import json
import os
import re


def determine_product_version_suffix(application):
    match = re.search(r"(?P<version>-\d+-\d+$)", application)
    if match:
        return match.group("version")
    return ""


def determine_snapshot_name(prefix, product_version):
    # The timestamp is added to the Snapshot name so that we can differentiate Snapshots from rebuilds of the same commit or tag.
    timestamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    return f"{prefix}{product_version}-{timestamp}".lower()


def parse_components_input(raw_input):
    return json.loads(raw_input)


def validate_component(component):
    assert (
        component["name"] != ""
        and component["containerImage"] != ""
        and component["revision"] != ""
        and component["repository"] != ""
    ), "Component must have component name, containerImage, revision and repository set."


def process_component(component, product_version_suffix):
    validate_component(component)
    return {
        "containerImage": component["containerImage"],
        "name": f"{component['name']}{product_version_suffix}",
        "source": {
            "git": {
                "revision": component["revision"],
                "url": component["repository"]
            }
        }
    }


def construct_snapshot(
    snapshot_name,
    pipeline_run_name,
    namespace,
    application,
    components
):
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
    application = os.environ["APPLICATION"]
    raw_components = parse_components_input(os.environ["COMPONENTS"])
    namespace = os.environ["NAMESPACE"]
    pipeline_run_name = os.environ["PIPELINE_RUN_NAME"]
    product_version = os.environ["PRODUCT_VERSION"]
    snapshot_name_result_path = os.environ["SNAPSHOT_NAME_RESULT_PATH"]

    product_version_suffix = determine_product_version_suffix(application)
    snapshot_name = determine_snapshot_name(application, product_version_suffix)
    components = [process_component(c, product_version_suffix) for c in raw_components]

    snapshot = construct_snapshot(
        snapshot_name=snapshot_name,
        pipeline_run_name=pipeline_run_name,
        namespace=namespace,
        application=application,
        components=components
    )

    write_snapshot(snapshot, snapshot_name_result_path)
    print("Rendered snapshot written to workspace.")
