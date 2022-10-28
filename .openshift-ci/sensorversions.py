#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re
import requests
import json

def isReleaseVersion(version):
    return bool(re.search(r"^\D+\d+\.\d+\.\d+$", version))

def filterQuayTags(tags):
    filteredtags = []
    for t in tags:
        name = t['name']
        if isReleaseVersion(name):
            filteredtags.append(name)
    return filteredtags

def queryQuayForTags():
    tags = []
    page = 1
    pageNotEmpty = True
    while pageNotEmpty:
        print("Going through page " + str(page))
        apiresponse = requests.get("https://quay.io/api/v1/repository/stackrox-io/main/tag/?page=" + str(page) + "&limit=100")
        rawtags = apiresponse.json()['tags']
        tags.extend(filterQuayTags(rawtags))
        if not bool(rawtags):
            pageNotEmpty = False
        page+=1
    return set(tags)

def transformTagsToNumbers(tags):
    numTags = []
    for t in tags:
        numTags.append(splitVersion(t))
    return sorted(numTags)

def splitVersion(version):
    digits = re.search(r"(\d+)\.(\d+)\.\D*(\d+)", version)
    return int(digits.group(1)), int(digits.group(2)), int(digits.group(3))

# TODO: grab current_image from os.environ["MAIN_IMAGE_TAG"] after manual testing is done
def getLastSensorVersionsFromQuay(current_version, num_versions):
    tags = queryQuayForTags()
    numericaltags = transformTagsToNumbers(tags)
    x,y,z = splitVersion(current_version)
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


def filterGitTags():
    apiresponse = requests.get("https://api.github.com/repos/stackrox/stackrox/git/refs/tags")
    rawtags = apiresponse.json()
    filteredtags = []
    for t in rawtags:
        name = t['ref']
        if isReleaseVersion(name):
            filteredtags.append(name)
    return set(filteredtags)

def jprint(obj):
    # create a formatted string of the Python JSON object
    text = json.dumps(obj, sort_keys=True, indent=4)
    print(text)


# TODO: grab current_image from os.environ["MAIN_IMAGE_TAG"] after manual testing is done
def getLastSensorVersionsFromGitTags(current_version, num_versions):
    tags = filterGitTags()
    numericaltags = transformTagsToNumbers(tags)
    x,y,z = splitVersion(current_version)
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
