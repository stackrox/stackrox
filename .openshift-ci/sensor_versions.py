#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re
import subprocess
from collections import defaultdict

def is_release_tag_git_cli(version):
    return bool(re.search(r"^\d+\.\d+\.\d+$", version))

def filter_git_cli_tags(rawtags):
    filteredtags = []
    for t in rawtags:
        if is_release_tag_git_cli(t):
            filteredtags.append(t)
    return set(filteredtags)

def cli_output_to_tags(stdoutput):
    separated = stdoutput.decode(encoding="utf-8").splitlines()
    return separated

def make_image_tag():
    return subprocess.check_output(["make", "tag"]).decode(encoding="utf-8")

def extract_y_from_main_image_tag(mainimagetag):
    return int(re.search(r"\d+\.(\d+)", mainimagetag).group(1))

def get_latest_tags(tags, num_versions):
    main_image_y = extract_y_from_main_image_tag(make_image_tag())
    top_patch_version = defaultdict(int)
    for t in tags:
        [major, minor, patch] = t.split('.')
        k = '.'.join([major, minor])
        if (int(minor) <= main_image_y):
            top_patch_version[k] = max(top_patch_version[k], int(patch))
    top_major_versions = sorted(list(top_patch_version.keys()), reverse=True)[:num_versions]
    return [t + '.' + str(top_patch_version[t]) for t in top_major_versions]

# get_last_sensor_versions_from_git_tags_cli gets the latest patches of the last num_versions major versions via Git CLI
def get_last_sensor_versions_from_git_tags_cli(num_versions):
    rawtags = cli_output_to_tags(subprocess.check_output(["git", "tag", "--list"]))
    tags = filter_git_cli_tags(rawtags)
    return get_latest_tags(tags, num_versions)
