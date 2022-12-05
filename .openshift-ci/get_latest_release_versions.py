#!/usr/bin/env python3

"""
Returns the latest patches of the last n major versions
"""

import sys
import re
import subprocess
from collections import defaultdict

cmd = "helm search repo stackrox-oss -l -o json | jq -r '.[] | select( .name==\"stackrox-oss/stackrox-secured-cluster-services\" ) | .version'"

add_repo_cmd = "helm repo add stackrox-oss https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource"
update_repo_cmd = "helm repo update"

def is_release_tag(version):
    return re.search(r"^\d+\.\d+\.\d+$", version) is not None

def filter_tags(rawtags):
    return set([t for t in rawtags if is_release_tag(t)])

def reduce_tags_to_latest_patch(tags):
    top_patch_version = defaultdict(int)
    major = "3"
    for t in tags:
        [minor, patch, _] = t.split('.')
        k = '.'.join([major, minor])
        top_patch_version[k] = max(top_patch_version[k], int(patch))
    top_major_versions = sorted(list(top_patch_version.keys()), reverse=True)
    return [t + "." + str(top_patch_version[t]) for t in top_major_versions]

def make_image_tag():
    return subprocess.check_output(["make", "--quiet", "--no-print-directory", "tag"]).decode(encoding="utf-8")

def extract_x_y_from_main_image_tag(tag):
    x_y = re.search(r"^(\d+)\.(\d+)", tag)
    return int(x_y.group(1)), int(x_y.group(2))

def get_latest_n_tags(tags, num_versions):
    central_major, central_minor = extract_x_y_from_main_image_tag(make_image_tag())
    tags_older_than_central = [t for t in tags if (int(t.split('.')[0]) < central_major or (int(t.split('.')[0]) == central_major and int(t.split('.')[1]) <= central_minor))]
    return tags_older_than_central[:num_versions]

# get_latest_release_versions gets the latest patches of the last num_versions major versions via Git CLI
def get_latest_release_versions(num_versions):
    rawtags = subprocess.check_output(cmd, shell=True).decode(encoding="utf-8").splitlines()
    tags = filter_tags(rawtags)
    latest_patch_tags = reduce_tags_to_latest_patch(tags)
    lastN = get_latest_n_tags(latest_patch_tags, num_versions)
    return map_to_helm_versions(lastN)

def map_to_helm_versions(versions):
    return [".".join(v.split(".")[1:]) + ".0" for v in versions]

def update_helm_repo():
    subprocess.run(add_repo_cmd, shell=True)
    print("stackrox-oss helm repository added")
    subprocess.run(update_repo_cmd, shell=True)
    print("stackrox-oss helm repository update")

def main(argv):
    n = int(argv[1]) if len(argv)>1 else 4
    update_helm_repo()
    latestversions = get_latest_release_versions(n)
    print(f"Last {n} versions:")
    print("\n".join(latestversions))

if (__name__ == "__main__"):
    main(sys.argv)
