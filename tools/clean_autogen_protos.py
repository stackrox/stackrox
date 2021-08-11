#!/usr/bin/env python3

import argparse
import pathlib


GENERATED_EXTENSIONS = ["pb.go", "pb.gw.go", "swagger.json"]


def find_files(path, fileglob):
    files_full = list(path.glob(fileglob))
    return files_full


def strip_path_extension(filelist):
    # We cannot use Path.stem directly as it doesn't handle double extensions (.pb.go) correctly
    files_extensionless = list(map(lambda f: (str(f).replace("".join(f.suffixes), "")), filelist))
    files_name_only = list(map(lambda f: pathlib.Path(f).stem, files_extensionless))
    return files_name_only


def find_difference(generated_list, proto_list):
    difference = set(generated_list) - set(proto_list)
    return difference


def filter_only_gen_files(candidates):
    return [x for x in candidates if any(str(x.name).endswith(extension) for extension in GENERATED_EXTENSIONS)]


def find_in_list(target_list, searchterms):
    searchterms = [f"{x}." for x in searchterms]  # Add a dot to only match full filenames
    return [x for x in target_list if any(str(x.name).startswith(term) for term in searchterms )]


def remove_files(target_list):
    for target in target_list:
        target.unlink()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--protos", type=pathlib.Path, help="Path to proto dir")
    parser.add_argument("--generated", type=pathlib.Path, help="Path to generated sources dir")
    v = parser.parse_args()

    proto_files = find_files(v.protos, "**/*.proto")
    generated_files = [f
                       for file_list in (find_files(v.generated, f'**/*.{ext}') for ext in GENERATED_EXTENSIONS)
                       for f in file_list]

    proto_stripped = strip_path_extension(proto_files)
    generated_stripped = strip_path_extension(generated_files)

    diff = find_difference(generated_stripped, proto_stripped)

    full_paths = find_in_list(generated_files, diff)
    final_diff = filter_only_gen_files(full_paths)

    if len(final_diff) > 0:
        print(f"Removing: {final_diff}")
        remove_files(final_diff)


if __name__ == '__main__':
    main()
