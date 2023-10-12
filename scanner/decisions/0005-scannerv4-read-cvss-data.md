# 0005 - Adapt Scanner V4 NVD CVSS data updating pipeline to integrate with NVD CVE Api

- **Author(s):** Yi Li <yli3@redhat.com>
- **Created:** [2023-10-10]

## Status

Accepted

## Context

The National Vulnerability Database (NVD) has announced the retirement of its JSON feeds by December 15, 2023. This creates the need to change our existing CVSS data updater pipeline for Scanner V4, which currently relies on these JSON feeds. 

The NVD JSON feed offers bundled data on a yearly basis, spanning from 2002 to the present year. A benefit is that only two requests are needed in the workflow for each year (one for the meta file and one for the data bundle). However, a limitation is the absence of filtering capabilities, such as by CVSS score or modification date. In contrast, the NVD API allows data retrieval through specific HTTP URL parameters, such as publish date, modified date, or CVSS score, enabling more precise data acquisition.


## Decision

The current NVD CVSS GitHub Workflow will be updated to fetch data from the NVD CVE API instead of NVD Json feeds while maintaining the freshness of the NVD CVSS data bundle in Google Storage. It will categorize the downloaded data by CVSS V3 severity levels, and these categorized data will be archived into individual zip files, such as `severity-low.zip`. The format of the JSON is the same as before.  

We've chosen to base our API requests on CVSS V3 severity to ensure all data integrated into Claircore possesses valid CVSS V3 metrics. Consequently, retrieving data without CVSS V3 information becomes irrelevant.

The CVSS Updater in Central will be modified to download zip files corresponding to four distinct severity levels and write them into a single zip file stored in Central's file system. This eliminates the need for JSON parsing in the updater. ScannerV4 will pull the zip bundle and populate the Matcher DB with the enrichment data.

## Consequences

One drawback of the NVD CVE API is its rate limit: only 5 requests are allowed per 30 seconds without an API Key. This restriction necessitates that our workflow fetch data at a slower pace. Additionally, transitioning from the NVD JSON feed to the NVD CVE API brings in pagination limitations. For example, when fetching data with a "low" severity, the API provides a maximum of 2000 CVEs per response. This means multiple requests are required to retrieve the complete dataset for "low" severity. The result is an increased number of smaller JSON files. Consequently, we need to compress all JSON files for a particular severity into a single zip archive.

Changing from the NVD JSON feed to the NVD CVE API also give us the conveniences of no need to filter out data with out valid CVSS V3 metrics, as mentioned above. 

Following this modification, Central will no longer have a dependency on Claircore. The Claircore enricher will be used in Scanner V4, which adopts Claircore as its primary scanning engine by design.

Per the NVD's documentation (https://nvd.nist.gov/General/News/change-timeline), the NVD CVE API omits CVSS v3 vector strings with a 'NONE' severity. This ensures we only obtain data bearing valid CVSS V3 metrics.