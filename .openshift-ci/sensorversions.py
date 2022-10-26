#!/usr/bin/env python3

"""
Returns the latest patches of the last 4 major versions
"""

import sys
import re

def splitVersion(version):
    digits = re.search(r"(\d+)\.(\d+)\.\D*(\d+)", version)
    return int(digits.group(1)), int(digits.group(2)), int(digits.group(3))

# TODO: figure out how to check if an image exists and check in this function
def imageExists(version):
    x, y, z = splitVersion(version)
    return (y > z)

# TODO: grab current_image from os.environ["MAIN_IMAGE_TAG"] after manual testing is done
def getLast4SensorVersions(current_version):
    x,y,z = splitVersion(current_version)
    latestversions = []
    for y_test in range(y-3,y+1):
        z_test = 0
        while imageExists("quay.io/stackrox-io/main:" + str(x) + "." + str(y_test) + "." + str(z_test+1)):
            z_test += 1
        latestversions.append(str(x) + "." + str(y_test) + "." + str(z_test))
    return latestversions

def main(argv):
    latestversions = getLast4SensorVersions(argv[1])
    print(str(latestversions[0]) + " " + str(latestversions[1]) + " " + str(latestversions[2]) + " " + str(latestversions[3]))

if (__name__ == "__main__"):
    main(sys.argv)
