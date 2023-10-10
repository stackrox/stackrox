# 0004 - Scanner V4 NVD CVSS enrichment data updating pipeline

- **Author(s):** Yi Li <yli3@redhat.com>, J. Victor Martins <jvdm@sdf.org>
- **Created:** [2023-09-11]

## Context

We are transitioning the StackRox Scanner to ClairCore as its primary scanning engine. To enrich vulnerabilities with CVSS information, ClairCore pulls CVSS scores from NVD during the Vulnerability Matching. However, Scanner V4 should not contact external endpoints other than registries and only contact Central for external data.

Our goal then are listed below:

1. Fetch NVD CVSS data that only contains available CVSS v3 scores from NVD CPE api and store in the Stackrox definitions bucket
2. Enable Central to retrieve NVD CVSS data from Stackrox definitions bucket.
3. Ensure seamless integration between Scanner V4 and Central, for enriching vulnerabilities with CVSS data.

## Decision

Central will have a new NVD CVSS Updater equipped by a novel enricher, based on ClairCore. It will download the NVD CVSS data from Stackrox definitions bucket and archive the files to one zip file. 

ClairCore will also be used in Scanner V4 CVSS data retriever (will be explained in detail below). It will require the archived NVD CVSS data bundle stored in Central. Central will handle the task of downloading and updating the NVD CVSS data bundle for Scanner V4 using ClairCore.

The NVD CVSS Updater will retrieve CVSS data from Google storage at a configurable interval. It will make a single HTTP call for each NVD data bundle categorized by CVSS V3 severity and startIndex, from the lowest data severity, to highest data severity, which is CRITICAL in CVSS V3. 

A GitHub workflow will keep the NVD data bundle in the Google Storage up-to-date. This Github workflow send http request to NVD CPE api to get data that only contains CVSS v3 metrics based on all levels of CVSS V3 severities. Such as "curl 'https://services.nvd.nist.gov/rest/json/cves/2.0?cvssV3Severity=CRITICAL&startIndex=0' > severity-critical-0.json". According to NVD site, the NVD CVE api does not contain CVSS v3 vector strings with a severity of NONE. So that makes sure we are only fetching data with valid CVSS V3 metrics.

This updater will download zip files corresponding to four distinct severity levels. Then it will write each json file from these zipped archives to a single zip file in Central's file system. This compressed file is typically around 50 MB in size.

The CVSS updater will operate as a GoRoutine, set to refresh the NVD CVSS data compressed file in Central at a configurable interval (default is 4 hours). By leveraging GoRoutine, this pipeline runs independently from Central, ensuring that any failures won't disrupt Central.

The CVSS Updater handler, paired with a singleton in Central, will offer an HTTP handler so Scanner can retrieve the consolidated NVD CVSS data.

Within Scanner V4, there's a component named as 'data retriever', which in Scanner V2 was called the 'updater'. We will rename to retriever because the name, 'updater', could be mistaken for the data updating GitHub action workflow in the Scanner V4 context. This CVSS data retriever in Scanner V4 communicates directly with Central's NVD CVSS data retrieval endpoint.

### Central CVSS data retrieving Endpoints

1. `/api/extensions/scanner-v4/definitions`: is configured to handle GET requests, which are processed by the NVD CVSS Updater handler in central.

## Consequences

This document focuses exclusively on the pipeline for updating CVSS enrichment data. 

We are not making any decisions for the updating pipeline of other data (vulnerability data and repo to cpe mapping data) at this moment.
