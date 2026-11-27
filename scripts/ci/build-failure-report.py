"""Write a single JUnit failure for a GitHub Actions build attempt."""

import json
import sys
import xml.etree.ElementTree as ET
from pathlib import Path


def main(jobs_file, output_dir):
    jobs = [json.loads(line) for line in jobs_file.read_text().splitlines()]
    if not jobs:
        raise SystemExit("No failed jobs returned for this attempt")
    suite = ET.Element("testsuite", name="Build", tests="1", failures="1")
    case = ET.SubElement(suite, "testcase", classname="Build", name="master build failed")
    failure = ET.SubElement(case, "failure", message="Master build failed")
    failure.text = "\n".join(
        f"{job['name']} — {', '.join(job['steps']) or 'No failed step recorded'} — {job['url']}"
        for job in jobs
    )
    output_dir.mkdir(parents=True, exist_ok=True)
    ET.ElementTree(suite).write(output_dir / "build.xml", encoding="utf-8", xml_declaration=True)


if __name__ == "__main__":
    main(Path(sys.argv[1]), Path(sys.argv[2]))
