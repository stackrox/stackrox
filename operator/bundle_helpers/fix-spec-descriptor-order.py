#!/usr/bin/env python3

import sys
import yaml


def fix_descriptor_order(descriptors):
    # Perform a stable sort (in Python, sorts are always stable) based on the path (excluding the last
    # component). This ensures items at the same nesting level retain their relative order, and children
    # will always come after their parents.
    # We add a `.` in front for simplicity, to ensure every key contains at least one dot. This will have
    # no impact on the output.
    descriptors.sort(key=lambda d: f'.{d["path"]}'.rsplit('.', 1)[0])


def allow_relative_field_dependencies(descriptors):
    for d in descriptors:
        x_descs = d.get('x-descriptors', [])
        for i, x_desc in enumerate(x_descs):
            if not x_desc.startswith('urn:alm:descriptor:com.tectonic.ui:fieldDependency:'):
                continue
            field, val = x_desc.split(':', 6)[-2:]
            if not field.startswith('.'):
                continue  # absolute path
            field = f'.{d["path"]}'.rsplit('.', 1)[0][1:] + field
            x_descs[i] = f'urn:alm:descriptor:com.tectonic.ui:fieldDependency:{field}:{val}'


def main():
    csv_doc = yaml.safe_load(sys.stdin)
    for crd in csv_doc['spec']['customresourcedefinitions']['owned']:
        descs = crd['specDescriptors']
        fix_descriptor_order(descs)
        allow_relative_field_dependencies(descs)
    print(yaml.safe_dump(csv_doc))


if __name__ == '__main__':
    main()
