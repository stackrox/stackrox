#!/usr/bin/env python3

import argparse
import pathlib


GENERATED_EXTENSIONS = sorted(
    [".pb.go", ".pb.gw.go", ".swagger.json"],
    key=len, reverse=True)


def find_files_rel(path, fileglob):
    files_full = set((str(f.relative_to(path)) for f in path.glob(fileglob)))
    return files_full


def proto_src(path):
    # GENERATED_EXTENSIONS are sorted in reverse order, this ensures the longest match wins
    for ext in GENERATED_EXTENSIONS:
        if path.endswith(ext):
            return path[:-len(ext)] + ".proto"
    return None


def remove_files(target_list):
    for target in target_list:
        target.unlink()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--protos", type=pathlib.Path, help="Path to proto dir")
    parser.add_argument("--generated", type=pathlib.Path, help="Path to generated sources dir")
    v = parser.parse_args()

    proto_files = find_files_rel(v.protos, "**/*.proto")
    generated_files = set().union(*(
        find_files_rel(v.generated, f'**/*{ext}') for ext in GENERATED_EXTENSIONS))

    to_remove = []
    for f in generated_files:
        if proto_src(f) not in proto_files:
            print(f"Removing {f} because {proto_src(f)} does not exist")
            to_remove.append(v.generated / f)

    if len(to_remove) > 0:
        remove_files(to_remove)


if __name__ == '__main__':
    main()
