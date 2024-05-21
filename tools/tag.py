#!/usr/bin/env python3

import sys

if len(sys.argv) < 2:
    print("Provide version string")
    sys.exit(1)

orig = sys.argv[1]

class Version:

    def __init__(self, major, minor, patch, build, commit):
        self.major = major
        self.minor = minor
        self.patch = patch
        self.build = build
        self.commit = commit

    @staticmethod
    def fromstring(s):
        major, minor, s = s.split(".", 2)
        patch, build, commit = s.split("-", 2)
        return Version(major, minor, patch, build, commit)

    def __str__(self):
        return f"{self.major}.{self.minor}.{self.patch}-{self.build}-{self.commit}"


if __name__ == "__main__":
    v = Version.fromstring(orig)
    v.minor = str(int(v.minor) + 1)
    v.patch = "0"
    print(v)
