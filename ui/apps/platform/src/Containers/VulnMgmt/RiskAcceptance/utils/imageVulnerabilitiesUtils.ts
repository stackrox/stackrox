/* eslint-disable import/prefer-default-export */
import { VulnerabilityRequest } from 'types/vuln_request.proto';
import { Vulnerability } from '../imageVulnerabilities.graphql';

export function combineVulnsWithVulnRequests(
    vulns: Vulnerability[],
    vulnRequests: VulnerabilityRequest[]
) {
    // create a map of cve->vulnRequest
    const cveToVulnRequestMap = vulnRequests.reduce((acc, vulnRequest) => {
        const cve = vulnRequest.cves.ids[0];
        acc[cve] = vulnRequest;
        return acc;
    }, {} as Record<string, VulnerabilityRequest>);
    // iterate through the vulns and add the vulnRequest if it exists for the specified cve
    const modifiedVulns =
        vulns.map((vuln) => {
            const modifiedVuln = { ...vuln };
            if (cveToVulnRequestMap[vuln.cve]) {
                modifiedVuln.vulnerabilityRequest = cveToVulnRequestMap[vuln.cve];
            }
            return modifiedVuln;
        }) || [];
    return modifiedVulns;
}
