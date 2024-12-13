#!/usr/bin/env python3

import json
import os
import time


def load_image_refs():
    return json.loads(os.getenv("IMAGE_REFS", '[]'))


def process_component(component, name_suffix):
    print(component)
    if name_suffix != "":
        name = f"{component["component"]}-{name_suffix}"
    else:
        name = component["component"]
    return {
        "containerImage": component["ref"],
        "name": name,
        "source": {
            "git": {
                "revision": component["revision"],
                "url": component["repository"]
            }
        }
    }


def construct_snapshot(snapshot_name_prefix, application, components):
    snapshot_name = f"{snapshot_name_prefix}-{int(time.time())}"
    return {
        "apiVersion": "appstudio.redhat.com/v1alpha1",
        "kind": "Snapshot",
        "metadata": {
            "name": snapshot_name
        },
        "spec": {
            "application": application,
            "components": components
        }
    }


def determine_component_name_suffix(application):
    return application.lstrip("acs-")


if __name__ == '__main__':
    application = os.getenv("APPLICATION", "")
    image_refs = load_image_refs()
    name_suffix = determine_component_name_suffix(application)
    components = [process_component(c, name_suffix) for c in image_refs]
    snapshot = construct_snapshot(
        f"tm-{application}",
        application,
        components
    )

    print("Snapshot:", snapshot)

    with open("snapshot.json", "w") as f:
        json.dump(snapshot, f)
