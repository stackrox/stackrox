#!/usr/bin/env python

"""
Fetch NVD data from the JSON 1.1 feed archives, parse them into API 2.0
schema and save into files.
"""

import gzip
import json
import logging
import os
from urllib import request
from urllib import parse


logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s: %(levelname)s: %(name)s %(message)s")


def convert_cve_feed_to_api(cve_feed):
    """Convert NVDCVEFeedJSON10DefCVEItem to CVEAPIJSON20DefCVEItem."""

    # Required fields.

    cve_api = {"id": cve_feed['cve']['CVE_data_meta']['ID'],
               "published": cve_feed['publishedDate'],
               "lastModified": cve_feed['lastModifiedDate'],
               "descriptions": [],
               "metrics": {}}

    # Description.

    for desc in cve_feed['cve'].get('description', {}).get('description_data', []):
        cve_api["descriptions"].append({"lang": desc["lang"],
                                        "value": desc["value"]})

    # Metrics.

    impact = cve_feed.get('impact', {})
    cvssMetricV2 = impact.get('baseMetricV2', {}).get('cvssV2', {})
    cvssMetricV3 = impact.get('baseMetricV3', {}).get('cvssV3', {})

    if cvssMetricV2:
        key = "cvssMetricV2"
        metric = {"type": "Primary",
                  "cvssData": {
                      "version": "2.0",
                      "vectorString": cvssMetricV2["vectorString"],
                      "baseScore": cvssMetricV2["baseScore"]}}
        cve_api["metrics"][key] = [metric]

    if cvssMetricV3:
        version = cvssMetricV3["version"]
        key = "cvssMetricV30" if version == "3.0" else "cvssMetricV31"
        metric = {"type": "Primary",
                  "cvssData": {
                      "version": version,
                      "vectorString": cvssMetricV3["vectorString"],
                      "baseScore": cvssMetricV3["baseScore"]}}
        cve_api["metrics"][key] = [metric]

    return {"cve": cve_api}


class LegacyLoader:

    BASE_URL = "https://nvd.nist.gov/feeds/json/cve/1.1/"

    log = logging.getLogger("LegacyLoader")

    def __init__(self, base_url=None):
        self._base_url = base_url or self.BASE_URL

    def fetch(self, year):
        url = parse.urljoin(self._base_url, f"nvdcve-1.1-{year}.json.gz")
        self.log.info("fetching and decompressing: %s", url)
        with request.urlopen(url, timeout=60) as resp:
            with gzip.GzipFile(fileobj=resp) as gz:
                cve_feed = json.load(gz)
        items = cve_feed.get("CVE_Items", [])
        self.log.info("cve count: %d", len(items))
        yield from (convert_cve_feed_to_api(c) for c in items)


def parse_args():
    import argparse
    from datetime import date

    parser = argparse.ArgumentParser()

    def dirpath(s):
        if os.path.isdir(s):
            return s
        raise ValueError

    parser.add_argument(
        "output",
        help="Path to directory where the NVD data will be saved.",
        type=dirpath)
    parser.add_argument(
        "--start-year",
        help="Fetch NVD data from the specified year.",
        type=int,
        default=2002)
    parser.add_argument(
        "--end-year",
        help="Fetch NVD data up to the specified year.",
        type=int,
        default=date.today().year)
    parser.add_argument(
        "--base-url",
        help="URL for the NVD CVE feed (version 1.1).")
    parser.add_argument(
        "--file-pattern",
        help="File name pattern of the NVD feed files.",
        default="%(year)s.nvd.json")

    return parser.parse_args()


def main(args):
    loader = LegacyLoader(args.base_url)
    for year in range(args.start_year, args.end_year + 1):
        path = os.path.join(args.output, args.file_pattern % {'year': year})
        logging.info("fetching '%d' to '%s'", year, path)
        with open(path, "w") as out:
            for cve in loader.fetch(year):
                json.dump(cve, out)


if __name__ == "__main__":
    main(parse_args())
