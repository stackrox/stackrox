#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re
import requests
import json

def isReleaseVersion(version):
    return bool(re.search(r"\d+\.\d+\.\d+$", version))

def filterTags(tags):
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
        tags.extend(filterTags(rawtags))
        if not bool(rawtags):
            pageNotEmpty = False
        page+=1
    return sorted(set(tags))

def transformTagsToNumbers(tags):
    numTags = []
    for t in tags:
        numTags.append(splitVersion(t))
    return numTags

def splitVersion(version):
    digits = re.search(r"(\d+)\.(\d+)\.\D*(\d+)", version)
    return int(digits.group(1)), int(digits.group(2)), int(digits.group(3))

# TODO: grab current_image from os.environ["MAIN_IMAGE_TAG"] after manual testing is done
def getLastSensorVersionsFromQuay(current_version, num_versions):
    tags = queryQuayForTags()
    numericaltags = transformTagsToNumbers(tags)
    print(numericaltags)
    print(numericaltags[::-1])

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
    latestversions = getLastSensorVersionsFromQuay(argv[1], 4)
    printversions = ""
    for version in latestversions:
        printversions += str(version) + " "
    print(printversions)


if (__name__ == "__main__"):
    main(sys.argv)
