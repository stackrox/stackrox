#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re
import requests
import json

def __isReleaseTagGit(version):
    return bool(re.search(r"^\D+\d+\.\d+\.\d+$", version))

def __isReleaseTagQuay(version):
    return bool(re.search(r"\d+\.\d+\.\d+$", version))

def __filterQuayTags(tags):
    filteredtags = []
    for t in tags:
        name = t['name']
        if __isReleaseTagQuay(name):
            filteredtags.append(name)
    return filteredtags

def __queryQuayForTags():
    tags = []
    page = 1
    pageNotEmpty = True
    while pageNotEmpty:
        print("Going through page " + str(page))
        apiresponse = requests.get("https://quay.io/api/v1/repository/stackrox-io/main/tag/?page=" + str(page) + "&limit=100")
        rawtags = apiresponse.json()['tags']
        tags.extend(__filterQuayTags(rawtags))
        if not bool(rawtags):
            pageNotEmpty = False
        page+=1
    return set(tags)

def __transformTagsToNumbers(tags):
    numTags = []
    for t in tags:
        numTags.append(__splitVersion(t))
    return sorted(numTags)

def __splitVersion(version):
    digits = re.search(r"(\d+)\.(\d+)\.\D*(\d+)", version)
    return int(digits.group(1)), int(digits.group(2)), int(digits.group(3))

def __filterGitTags():
    apiresponse = requests.get("https://api.github.com/repos/stackrox/stackrox/git/refs/tags")
    rawtags = apiresponse.json()
    filteredtags = []
    for t in rawtags:
        name = t['ref']
        if __isReleaseTagGit(name):
            filteredtags.append(name)
    return set(filteredtags)

# TODO: grab current_image from os.environ["MAIN_IMAGE_TAG"] after manual testing is done
# getLastSensorVersionsFromQuay gets the latest patches of the last num_versions major versions
# querying quay API is slow, prefer using git tag API
def getLastSensorVersionsFromQuay(current_version, num_versions):
    tags = __queryQuayForTags()
    numericaltags = __transformTagsToNumbers(tags)
    x,y,z = __splitVersion(current_version)
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

# TODO: grab current_image from os.environ["MAIN_IMAGE_TAG"] after manual testing is done
# getLastSensorVersionsFromGitTags gets the latest patches of the last num_versions major versions
# much faster than querying quay
def getLastSensorVersionsFromGitTags(current_version, num_versions):
    tags = __filterGitTags()
    numericaltags = __transformTagsToNumbers(tags)
    x,y,z = __splitVersion(current_version)
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

def main(argv):
    latestversions = getLastSensorVersionsFromGitTags(argv[1], 4)
    #latestversions = getLastSensorVersionsFromQuay(argv[1], 4)
    printversions = ""
    for version in latestversions:
        printversions += str(version) + " "
    print(printversions)

if (__name__ == "__main__"):
    main(sys.argv)
