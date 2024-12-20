#!/usr/bin/env python3

import datetime
import json
import os


def parse_image_refs(image_refs):
    return json.loads(image_refs)


def validate_component(component):
    assert (
        component["name"] != ""
        and component["containerImage"] != ""
        and component["revision"] != ""
        and component["repository"] != ""
    ), "Component must have component name, ref, revision and repository set. Check container image labels."


def process_component(component, name_suffix):
    validate_component(component)
    if name_suffix != "":
        name = f"{component['component']}-{name_suffix}"
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
    snapshot_name = f"{snapshot_name_prefix}_{snapshot_version_suffix}-{timestamp}"
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


def determine_component_version_suffix(application):
    # TODO: this as a regex
    return application.lstrip("acs-")


def sanitize_tag(tag):
    return tag.replace(".", "-")


if __name__ == '__main__':
    application = os.environ["APPLICATION"]
    pipeline_run_name = os.environ["PIPELINE_RUN_NAME"]
    namespace = os.environ["NAMESPACE"]
    image_refs = parse_image_refs(os.environ["IMAGE_REFS"])
    snapshot_version_suffix = sanitize_tag(os.environ["MAIN_IMAGE_TAG"])
    name_suffix = determine_component_version_suffix(application)
    components = [process_component(c, name_suffix) for c in image_refs]
    snapshot = construct_snapshot(
        snapshot_name_prefix=application,
        snapshot_version_suffix=snapshot_version_suffix,
        pipeline_run_name=pipeline_run_name,
        namespace=namespace,
        application=application,
        components=components
    )

    with open("snapshot.json", "w") as f:
        json.dump(snapshot, f)

    print(snapshot["metadata"]["name"], end="")
