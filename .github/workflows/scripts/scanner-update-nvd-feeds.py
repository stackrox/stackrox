#!/usr/bin/env python

"""
Fetch NVD data from the JSON 2.0 feed archives and save into files.
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

DEFAULT_BASE_URL = "https://nvd.nist.gov/feeds/json/cve/2.0/"

class FeedLoader:

    log = logging.getLogger("FeedLoader")

    def __init__(self, base_url):
        self._base_url = base_url

    def fetch(self, year):
        url = parse.urljoin(self._base_url, f"nvdcve-2.0-{year}.json.gz")
        self.log.info("fetching and decompressing: %s", url)
        with request.urlopen(url, timeout=60) as resp:
            with gzip.GzipFile(fileobj=resp) as gz:
                data = json.load(gz)
        yield from (v for v in data["vulnerabilities"]
        if v["cve"]["vulnStatus"].lower() != "rejected")


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
        help="URL for the NVD CVE feed (version 2.0).",
        default=DEFAULT_BASE_URL)
    parser.add_argument(
        "--file-pattern",
        help="File name pattern of the NVD feed files.",
        default="%(year)s.nvd.json")

    return parser.parse_args()


def main(args):
    loader = FeedLoader(args.base_url)
    for year in range(args.start_year, args.end_year + 1):
        path = os.path.join(args.output, args.file_pattern % {'year': year})
        logging.info("fetching '%d' to '%s'", year, path)
        with open(path, "w") as out:
            for cve in loader.fetch(year):
                json.dump(cve, out)


if __name__ == "__main__":
    main(parse_args())
