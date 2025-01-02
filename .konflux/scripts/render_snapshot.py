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
    timestamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    return f"{prefix}{product_version}-{timestamp}".lower()


def parse_image_refs(image_refs):
    return json.loads(image_refs)


def validate_component(component):
    assert (
        component["name"] != ""
        and component["containerImage"] != ""
        and component["revision"] != ""
        and component["repository"] != ""
    ), "Component must have component name, ref, revision and repository set."


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
    application = os.environ["APPLICATION"] # 1
    product_version_suffix = determine_product_version_suffix(application)
    snapshot_name = determine_snapshot_name(application, product_version_suffix)
    image_refs = parse_image_refs(os.environ["IMAGE_REFS"]) # 2
    components = [process_component(c, product_version_suffix) for c in image_refs]

    product_version = os.environ["PRODUCT_VERSION"] # 3
    pipeline_run_name = os.environ["PIPELINE_RUN_NAME"] # 4
    namespace = os.environ["NAMESPACE"] # 5

    snapshot = construct_snapshot(
        snapshot_name=snapshot_name,
        pipeline_run_name=pipeline_run_name,
        namespace=namespace,
        application=application,
        components=components
    )

    snapshot_name_result_path = os.environ["SNAPSHOT_NAME_RESULT_PATH"]
    write_snapshot(snapshot, snapshot_name_result_path)
    print("Rendered snapshot written to workspace.")
