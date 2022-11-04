#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re
import subprocess

def isReleaseTagGitCLI(version):
    return bool(re.search(r"^\d+\.\d+\.\d+$", version))

def filterGitCLITags(rawtags):
    filteredtags = []
    for t in rawtags:
        if isReleaseTagGitCLI(t):
            filteredtags.append(t)
    return set(filteredtags)

def cliOutputToTags(stdoutput):
    separated = stdoutput.decode(encoding="utf-8").splitlines()
    return separated

def transformTagsToNumbers(tags):
    numTags = []
    for t in tags:
        numTags.append(splitVersion(t))
    return sorted(numTags)

def splitVersion(version):
    digits = re.search(r"(\d+)\.(\d+)\.\D*(\d+)", version)
    return int(digits.group(1)), int(digits.group(2)), int(digits.group(3))

def getLatestTags(current_version, tags, num_versions):
    numericaltags = transformTagsToNumbers(tags)
    _,y,_ = splitVersion(current_version)
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

# getLastSensorVersionsFromGitTagsCLI gets the latest patches of the last num_versions major versions via Git CLI
def getLastSensorVersionsFromGitTagsCLI(current_version, num_versions):
    rawtags = cliOutputToTags(subprocess.check_output(["git", "tag", "--list"]))
    tags = filterGitCLITags(rawtags)
    return getLatestTags(current_version, tags, num_versions)
