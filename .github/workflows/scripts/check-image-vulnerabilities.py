import argparse
import http
import json
import os

import requests

QUAY_ORG="rhacs-eng"

VULNERABILITY_SEVERITIES = {
    "Unknown": 0,
    "Negligible": 1,
    "Low": 2,
    "Medium": 3,
    "High": 4,
    "Critical": 5,
}

def wrap_quay_api(image, endpoint, query_params):
    BEARER_TOKEN=os.getenv("QUAY_BEARER_TOKEN")
    if not BEARER_TOKEN:
        raise Exception("No QUAY_BEARER_TOKEN provided.")

    url = f"https://quay.io/api/v1/repository/{QUAY_ORG}/{image}/{endpoint}"
    headers = {"Authorization": f"Bearer {BEARER_TOKEN}"}

    r = requests.get(url=url, params=query_params, headers=headers)
    if not r.status_code == http.HTTPStatus.OK:
        raise Exception("Failed request to Quay:", r.status_code, r.text)
    return r.json()


def find_manifest(image, tag):
    tag_data = wrap_quay_api(image, "tag", {"onlyActiveTags": "true", "specificTag": tag, "limit": 1})
    number_of_tags = len(tag_data["tags"])
    if number_of_tags != 1:
        raise Exception(f"Failed to identify tag - {number_of_tags} tag(s) returned from Quay for {image}:{tag}.")

    return tag_data["tags"][0]["manifest_digest"]


def check_vulnerabilities(image, manifest):
    vuln_data = wrap_quay_api(image, f"manifest/{manifest}/security", {"vulnerabilities": "true"})
    scan_status = vuln_data["status"]
    if scan_status != "scanned":
        raise Exception(f"Image '{image}' with manifest {manifest} not scanned yet - current status: {scan_status}.")

    packages = vuln_data["data"]["Layer"]["Features"]
    vuln_report = []
    for p in packages:
        if len(p["Vulnerabilities"]) > 0:
            vuln_report.append(collect_vulnerability_information(p))
    return vuln_report


def collect_vulnerability_information(package):
    package_information = {
        "name": package["Name"],
        "version": package["Version"],
        "vulnerabilities": [],
    }
    for vuln in package["Vulnerabilities"]:
        package_information["vulnerabilities"].append({
            "name": vuln["Name"],
            "severity": vuln["Severity"],
            "link": vuln["Link"],
            "severity_level": VULNERABILITY_SEVERITIES.get(vuln["Severity"], -1)
        })

    return package_information


def get_vulnerability_report(image):
    manifest = find_manifest(image["name"], image["tag"])
    return check_vulnerabilities(image["name"], manifest)


def dump_report(images, as_json=False):
    if as_json:
        print(json.dumps(images))
    else:
        print("{:<30} {:<10} {:<20} {:<20} {:<80}".format(
            "IMAGE",
            "SEVERITY", "PACKAGE NAME",
            "PACKAGE VERSION", "VULNERABILITY",
        ))
        print("=" * 185)

        output = []
        for i in images["images"]:
            if len(i["vulnerable_packages"]) > 0:
                for package in i["vulnerable_packages"]:
                    for vuln in package["vulnerabilities"]:
                        output.append({
                            "image": i["name"], "vuln_name": vuln["name"], "vuln_severity": vuln["severity"],
                            "vuln_severity_level": vuln["severity_level"],
                            "package_name": package["name"], "package_version": package["version"]
                        })


        output.sort(key=lambda x: (-x["vuln_severity_level"], x["image"], x["package_name"]))
        for x in output:
            print("{:<30} {:<10} {:<20} {:<20} {:<80}".format(
                x["image"], x["vuln_severity"],
                x["package_name"], x["package_version"], x["vuln_name"],
            ))


def main():
    parser = argparse.ArgumentParser(description='Fetch vulnerability information from Quay for a release (candidate).')
    parser.add_argument("tag")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    # Vulnerability information is attached to the child manifest, suffix -amd64
    tag = f"{args.tag}-amd64"
    images = {
        "images": [
            {"name": "central-db", "tag": tag},
            {"name": "collector", "tag": tag},
            {"name": "collector-slim", "tag": tag},
            {"name": "main", "tag": tag},
            {"name": "roxctl", "tag": tag},
            {"name": "scanner", "tag": tag},
            {"name": "scanner-db", "tag": tag},
            {"name": "scanner-db-slim", "tag": tag},
            {"name": "scanner-slim", "tag": tag},
            {"name": "stackrox-operator", "tag": tag},
        ]
    }

    for image in images["images"]:
            image["vulnerable_packages"] = get_vulnerability_report(image)

    dump_report(images, args.json)

if __name__ == "__main__":
    main()
