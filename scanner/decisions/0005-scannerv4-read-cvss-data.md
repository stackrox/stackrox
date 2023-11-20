# 0005 - Adapt Scanner V4 NVD CVSS data updating pipeline to integrate with NVD CVE Api

- **Author(s):** Yi Li <yli3@redhat.com>
- **Created:** [2023-10-10]

## Context
We are transitioning the Scanner V4 CVSS data update pipeline from the NVD JSON feeds to the NVD CVE API. This change is driven by NVD's announcement to retire their JSON feeds on December 15, 2023.

Our goal then are listed below:

1. Fetch NVD CVSS data that only contains valid CVSS v3 scores from NVD CVE api and store in the Stackrox definitions Google bucket.
2. Enable Central to retrieve NVD CVSS data from Google bucket.
3. Ensure seamless integration between Scanner V4 and Central, for enriching vulnerabilities with CVSS data.

## Decision

Central will have a new NVD CVSS Updater. It will download the NVD CVSS data from Google bucket and archive the json files to one single zip file.

ClairCore will also be used in Scanner V4 CVSS data retriever (will be explained in detail below). It will require the archived NVD CVSS data bundle stored in Central. Central will handle the task of downloading and updating the NVD CVSS data bundle for Scanner V4 using ClairCore.

The NVD CVSS Updater will retrieve CVSS data from Google storage at a configurable interval. It will make a single HTTP call for each NVD data bundle categorized by CVSS V3 severity. As an illustration, all CVE data with a CVSS V3 severity of 'low' will be archived into 'severity-low.zip' in Google storage and then being downloaded by the updater.

A GitHub workflow will keep the NVD data bundle in the Google Storage up-to-date. This Github workflow send http request to NVD CVE api to get data based on each level of CVSS V3 severities. Such as "curl 'https://services.nvd.nist.gov/rest/json/cves/2.0?cvssV3Severity=CRITICAL&startIndex=0' > severity-critical-0.json".  Per the NVD's documentation, the NVD CVE API omits CVSS v3 vector strings with a 'NONE' severity. This ensures we only obtain data bearing valid CVSS V3 metrics. For JSON files of a singular severity, we compress them into a single zip file, such as severity-low.zip.

This updater will download zip files corresponding to four distinct severity levels. Then it will write each json file from these zipped archives to a single zip file in Central's file system. We will not need to do any json parsing in the updater. This compressed file is typically around 50 MB in size.

The CVSS updater will operate as a GoRoutine, set to refresh the NVD CVSS data compressed file in Central at a configurable interval (default is 4 hours). By leveraging GoRoutine, this pipeline runs independently from Central, ensuring that any failures won't disrupt Central.

The CVSS Updater handler, paired with a singleton in Central, will offer an HTTP handling so Scanner can retrieve the NVD CVSS data.

Within Scanner V4, there's a component named as 'data retriever', which in Scanner V2 was called the 'updater'. We will rename to retriever because the name, 'updater', could be mistaken for the data updating GitHub action workflow in the Scanner V4 context. This CVSS data retriever in Scanner V4 communicates directly with Central's NVD CVSS data retrieval endpoint.

### Central CVSS data retrieving Endpoints

This decision remains the same in the previous ADR: https://github.com/stackrox/stackrox/blob/master/scanner/decisions/0004-scannerv4-read-vulneralbilities.md

## Consequences

This document focuses exclusively on the pipeline for updating CVSS enrichment data. 

We are not making any decisions for the updating pipeline of other data (vulnerability data and repo to cpe mapping data) at this moment.
