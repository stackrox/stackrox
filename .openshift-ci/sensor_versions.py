#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re
import subprocess

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

def transform_tags_to_numbers(tags):
    numTags = []
    for t in tags:
        numTags.append(split_version(t))
    return sorted(numTags)

def split_version(version):
    digits = re.search(r"(\d+)\.(\d+)\.(\d+)", version)
    return int(digits.group(1)), int(digits.group(2)), int(digits.group(3))

def extract_y_from_main_image_tag(mainimagetag):
    return int(re.search(r"\d+\.(\d+)", mainimagetag).group(1))

def get_latest_tags(tags, num_versions):
    numericaltags = transform_tags_to_numbers(tags)
    y = extract_y_from_main_image_tag(make_image_tag())
    latestversions = []
    ycurr = y
    for tags in numericaltags[::-1]:
        if tags[1] < y-num_versions+1:
            break
        if tags[1] < ycurr:
            ycurr = tags[1]
        if tags[1] == ycurr:
            latestversions.append(str(tags[0]) + "." + str(tags[1]) + "." + str(tags[2]))
            ycurr-=1
    return latestversions

def make_image_tag():
    return subprocess.check_output(["make", "tag"]).decode(encoding="utf-8")

# get_last_sensor_versions_from_git_tags_cli gets the latest patches of the last num_versions major versions via Git CLI
def get_last_sensor_versions_from_git_tags_cli(num_versions):
    rawtags = cli_output_to_tags(subprocess.check_output(["git", "tag", "--list"]))
    tags = filter_git_cli_tags(rawtags)
    return get_latest_tags(tags, num_versions)
