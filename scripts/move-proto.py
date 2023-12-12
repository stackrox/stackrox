#!/usr/bin/env python3

import os
import shutil

staging = "proto/staging"

shutil.rmtree(staging)
os.mkdir(staging)

def processFile(storageFile, stagingFile):
    for line in storageFile.readlines():
        line = line.replace("package storage;", "package v1;")
        line = line.replace('option go_package = "storage";', 'option go_package = "v1";')
        line = line.replace('option java_package = "io.stackrox.proto.storage";', 'option java_package = "io.stackrox.proto.api.v1";')
        line = line.replace('import "storage/', 'import "api/v1/')
        line = line.replace("storage.", "v1.")

        if "scrub" not in line and "validate" not in line and "[(gogoproto.moretags)" in line:
            gogoIndex = line.index("[(gogoproto.moretags)")
            if line[gogoIndex-1] == " ":
                line = line[:gogoIndex-1]
            else:
                line = line[:gogoIndex]
            line += ";\n"
        stagingFile.write(line)

for root, dirs, files in os.walk("proto/storage", topdown=False):
   for name in files:
      if not name.endswith(".proto"):
        continue
      with open(staging+"/"+name, "w") as f:
          processFile(open(os.path.join(root, name)), f)
