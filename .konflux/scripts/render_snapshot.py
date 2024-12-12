#!/usr/bin/env python3

import json
import os
import time


def load_image_refs():
    return json.loads(os.getenv("IMAGE_REFS", "[]"))


def process_component(component):
    return {
        "containerImage": component["ref"],
        "name": component["component"],
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


if __name__ == '__main__':
    application = os.getenv("APPLICATION", "")
    image_refs = load_image_refs()
    components = [process_component(c) for c in image_refs]
    snapshot = construct_snapshot(
        f"tm-{application}",
        application,
        components
    )

    print("Snapshot:", snapshot)

    with open("snapshot.json", "w") as f:
        json.dump(snapshot, f)
