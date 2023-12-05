# 0006 - Integration of Red Hat CVSS Scores

- **Author(s):** J. Victor Martins <jvdm@sdf.org>
- **Created:** [2023-12-01 Fri]

## Status

Accepted.

## Context

ClairCore, the scanning engine of StackRox Scanner V4, can pull and match CVSS scores.  These scores are retrieved exclusively from NVD.  However, customers expect Red Hat's vulnerability scores when it comes to Red Hat images and components.  These scores are provided by Red Hat security sources; the most current and complete in terms of metadata and product stream information is CSAF/VEX.

The Vulnerability Report contains CVSS scores in the [Enrichments field][1].  [Enrichers][2] are responsible for managing this field.  The field is a table, mapping [enricher type to a list of enrichment results][3].  For CVSS, these are [JSON objects mapping vulnerability IDs to a list of CVSS scores][4].  Enrichers play a role during "vulnerability updating" and "vulnerability matching".  They populate an enrichment table in the database during updates and use that information to populate enrichments during matching.

[1]: https://github.com/quay/claircore/blob/faffb8e880263171ca9b54dc2f5609547e53cbb7/vulnerabilityreport.go#L23
[2]: https://github.com/quay/claircore/blob/faffb8e880263171ca9b54dc2f5609547e53cbb7/libvuln/driver/enrichment.go#L75
[3]: https://github.com/quay/claircore/blob/faffb8e880263171ca9b54dc2f5609547e53cbb7/internal/matcher/match.go#L184-L191
[4]: https://github.com/quay/claircore/blob/faffb8e880263171ca9b54dc2f5609547e53cbb7/enricher/cvss/cvss.go#L269

Currently, there is only one enricher, "clair.cvss".  It offers maps from vulnerability ID to CVSS scores pulled from NVD.

![Diagram depicting the vulnerability scanning process in the "Enriched Matching" phase. It includes "Vulnerability Matchers" for different operating systems, a central "Match()" process, and parallel "Vulnerability Collector" and "Enrichment Collector" processes. The output is an "Index Report".](images/claircore-matching.svg)

The Enrichers and Vulnerability Matchers are distinct objects, but they perform similar functions in the scanning process. The workers involved in the matching steps work together to call both, passing on the results to collectors.  These collectors are responsible for populating the final results in the vulnerability report.  See [the enriched matching source code](https://github.com/quay/claircore/blob/9cca6fecc233483e0435978715173a151a8455e1/internal/matcher/match.go#L92) for details.

### Challenges in supporting RH CVSS scores

A natural step to enrich vulnerability reports with this framework is to create an RH CVSS Score enricher, e.g., "rhel.cvss".  But, different from NVD, Red Hat Security scores are tied to the product streams and the version of the components affected.  This brings some considerations:

1.  RH CVSS scores would require parsing and storing additional details, such as product IDs or CPEs, to identify the product stream and ID.  This is a significant change compared to how enrichment happens for NVD CVSS matching, which is a  [simple match of CVE IDs in the vulnerability object](https://github.com/quay/claircore/blob/faffb8e880263171ca9b54dc2f5609547e53cbb7/enricher/cvss/cvss.go#L276-L284).  This amount of information can be significant (e.g., hundreds of KiB per CVE), so some pre-processing might be necessary to avoid storing all the data.
2.  Red Hat security does offer rich CVSS score information and product stream relationships through CSAF/VEX endpoints.  However, the fact that vulnerability updaters and enrichers do not currently coordinate brings additional complexity to leveraging the same information for both workflows.  Doing that would help us avoid additional round-trips to the security sources.  Also, it could allow us to leverage the vulnerability information in the DB to select the CVSS scores to add.
3.  StackRox Scanner cannot reach out to the security sources directly.  It relies on ClairCore's "air-gapped" settings and creates bundles with security data consumed through a separate update channel.  Additional complexity exists to ensure CI updaters generate backward-compatible bundles.  The data size also plays a role since it may impact the system IO when fetching and updating the DB.
4.  ClairCore's future integration with CSAF/VEX may make current integration efforts redundant in the long run.  How that integration will work currently needs to be clarified.

## Decision

We will create an out-of-tree ClairCore Enricher (i.e., not part of ClairCore) for RH CVSS scores.  The enricher will pull RH CVSS scores and vulnerability metadata from Red Hat Security VEX endpoints.  It is the most up-to-date security source and offers product stream information, which allows for the specific assignment of scores to vulnerabilities based on the component's repository and distribution.

The RH Enricher will fetch every CVE document in the VEX directory, parse the relevant information, and create entries in the enrichment table with only the necessary information for enriching steps:

1.  The enricher creates a map from CPE to Name for each `product_name` category in the `"product_tree"` object. This map helps the matcher identify the vulnerability product. The CPE is available as an identifier helper.  Example `cpe:/o:redhat:enterprise_linux:9::baseos -> BaseOS-9.1.0.Z.MAIN`:
    ```
    {
      "category": "product_name",
      "name": "Red Hat Enterprise Linux BaseOS (v. 9)",
      "product": {
        "name": "Red Hat Enterprise Linux BaseOS (v. 9)",
        "product_id": "BaseOS-9.1.0.Z.MAIN",
        "product_identification_helper": {
          "cpe": "cpe:/o:redhat:enterprise_linux:9::baseos"
        }
      }
    }
    ```
2.  Enricher will also store the list of scores and the map of products they "affect" (or are relevant for).
3.  The information will be stored in the `enrichment` table within the current schema:
    | ID | updater   | tags                                 | data                                   |
    |----|-----------|--------------------------------------|----------------------------------------|
    |  1 | rhel.cvss | [CVE-YYYY-NNNN, RHSA-YYYY:XXXX, ...] | {"scores": ..., "product_stream": ...} |

    The RH Enricher will match enrichments by parsing the package name, version, and IDs for every vulnerability.  IDs entail not only CVE but also advisories.  The querying should return the enrichment data whenever the vulnerability references any of those IDs.  Further processing will identify if the product stream and ID match the score specification.

    In pseudo-code:

    ```bash
    idRegexp = # regex that matches CVE IDs and RH advisories (e.g., RHSA)
    vr       = # The Vulnerability Report
    db       = # The enrichment database connection

    enrichments = []

    for pkg in the $vr.Packages:
        vulnsInPkg = vr.PackageVulnerabilities.Get($pkg.ID)
        for $vuln in $vulnsInPkg:
            ids = idRegexp.Match($vuln.Name + $vuln.Description + $vuln.Links)
            for e in $db.QueryEnrichments($ids):
                pid = e.ProductMap[$vuln.Repo.CPE]
                if e.Product == "%s:%s:%s.%s".format($pid, $pkg.Name, $pkg.Version, $pkg.Arch):
                    enrichments.append(e)
    return enrichments
    ```

To achieve the above, changes will be made to the Offline Importer `jsonblob` storage to save and load enrichments and vulnerabilities in the same bundle.

We will run the `rhel.cvss` enricher in CI, together with the other updaters, and create a bundle per StackRox Scanner release with the enrichments.

The API will include a new field in the vulnerability proto called "Scores."  That field will contain a map of enricher IDs to an array of scores.  Each score will contain the score information, such as the vector string and the score version.  Example:

```protobuf
  message Score {
    vectorString string = 1;
    version = 2;
  }
```

The API will return both NVD CVSS scores and RH CVSS scores, and clients will consume that information as needed.

## Consequences

1.  CSAF/VEX is the most comprehensive Red Hat security data source recommended by Red Hat Security.  We will benefit from the most up-to-date and accurate information. 
2.  No need for a separate CVSS updater pipeline (CI workflow, handlers in Central, etc).  All the information will be bundled with the vulnerabilities, with versions per StackRox release.
2.  Using a separate out-of-three enricher breaks a dependency on future CSAF/VEX work integration in StackRox Scanner.  That helps StackRox Scanner move faster, but it might be redundant work that can be discarded once ClairCore adopts CSAF/VEX.
3.  ClairCore does not offer dedicated fields to reference the vulnerability ID (CVE) and related advisories.  Hence, we are parsing that information from multiple domains (name, description, and links).  That opens the door for inaccuracies since that information might include unrelated IDs.
4.  How ClairCore plans to adopt CSAF/VEX security data needs to be clarified.  With an RH CVSS Score Enricher, two round trips to pull the CSAF/VEX information would be required in the current framework.
5.  We are optimizing the enrichment bundle and DB size by storing only the necessary information for enriching RH components instead of keeping the whole VEX document in the enrichment database.  Each VEX document contains hundreds of KiB, counting roughly ~900MiB.  On the other hand, future changes and needed information might require modifications to the RH enrichment schema.  Since we are planning to bundle enrichments with vulnerability updates per StackRox Scanner release, we will have bundles per release, which will give us the ability to control the backward compatibility of changes.
