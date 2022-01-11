import { gql } from '@apollo/client';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { Scope, VulnerabilityRequest } from 'types/vuln_request.proto';

// This type is specific to the way we query using GraphQL
export type Vulnerability = {
    id: string;
    cve: string;
    isFixable: boolean;
    severity: VulnerabilitySeverity;
    cvss: string;
    scoreVersion: string;
    discoveredAtImage: string;
    components: EmbeddedImageScanComponent[];
    vulnerabilityRequest?: VulnerabilityRequest;
};

export type VulnerabilityWithRequest = Vulnerability & {
    vulnerabilityRequest: VulnerabilityRequest;
};

// This type is specific to the way we query using GraphQL
export type EmbeddedImageScanComponent = {
    id: string;
    name: string;
    version: string;
    fixedIn: string;
};

export type GetImageVulnerabilitiesData = {
    image: {
        name: {
            registry: string;
            remote: string;
            tag: string;
        };
        vulnCount: number;
        vulns: Vulnerability[];
    };
};

export type GetImageVulnerabilitiesVars = {
    imageId: string;
    vulnsQuery: string;
    // @TODO: vulnerabilityRequests.graphql also uses this pagination structure. We should refactor this
    pagination: {
        limit: number;
        offset: number;
        sortOption: {
            field: string;
            reversed: boolean;
        };
    };
};

export const GET_IMAGE_VULNERABILITIES = gql`
    query getImageVulnerabilities($imageId: ID!, $vulnsQuery: String, $pagination: Pagination) {
        image(id: $imageId) {
            name {
                registry
                remote
                tag
            }
            vulnCount(query: $vulnsQuery)
            vulns(query: $vulnsQuery, pagination: $pagination) {
                id: cve
                cve
                isFixable
                severity
                scoreVersion
                cvss
                discoveredAtImage
                components {
                    id
                    name
                    version
                    fixedIn
                }
            }
        }
    }
`;

export type DeferVulnerabilityRequest = {
    cve: string;
    comment: string;
    scope: Scope;
    expiresWhenFixed?: boolean;
    expiresOn?: string;
};

export const DEFER_VULNERABILITY = gql`
    mutation deferVulnerability($request: DeferVulnRequest!) {
        deferVulnerability(request: $request) {
            id
        }
    }
`;

export type MarkFalsePositiveRequest = {
    cve: string;
    comment: string;
    scope: Scope;
};

export const MARK_FALSE_POSITIVE = gql`
    mutation markVulnerabilityFalsePositive($request: FalsePositiveVulnRequest!) {
        markVulnerabilityFalsePositive(request: $request) {
            id
        }
    }
`;
