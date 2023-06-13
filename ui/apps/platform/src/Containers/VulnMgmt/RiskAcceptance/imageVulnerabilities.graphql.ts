import { gql } from '@apollo/client';

import { VulnerabilitySeverity, VulnerabilityState } from 'types/cve.proto';
import { SlimUser } from 'types/user.proto';
import { DeferralRequest, RequestComment, RequestStatus, Scope } from 'types/vuln_request.proto';

export type VulnerabilityRequest = {
    id: string;
    targetState: VulnerabilityState;
    status: RequestStatus;
    expired: boolean;
    requestor: SlimUser;
    approvers: SlimUser[];
    comments: RequestComment[];
    scope: Scope;
    deferralReq: DeferralRequest;
    updatedDeferralReq: DeferralRequest;
    cves: {
        cves: string[];
    };
};

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
    vulnerabilityRequest: VulnerabilityRequest;
};

// This type is specific to the way we query using GraphQL
export type EmbeddedImageScanComponent = {
    id: string;
    name: string;
    version: string;
    fixedIn: string;
    dockerfileLine?: {
        line: number;
        instruction: string;
        value: string;
    };
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
            vulnCount: imageVulnerabilityCount(query: $vulnsQuery)
            vulns: imageVulnerabilities(query: $vulnsQuery, pagination: $pagination) {
                id
                cve
                isFixable
                severity
                scoreVersion
                cvss
                discoveredAtImage
                components: imageComponents {
                    id
                    name
                    version
                    fixedIn
                }
                vulnerabilityRequest: effectiveVulnerabilityRequest {
                    id
                    targetState
                    status
                    expired
                    requestor {
                        id
                        name
                    }
                    approvers {
                        id
                        name
                    }
                    comments {
                        createdAt
                        id
                        message
                        user {
                            id
                            name
                        }
                    }
                    deferralReq {
                        expiresOn
                        expiresWhenFixed
                    }
                    updatedDeferralReq {
                        expiresOn
                        expiresWhenFixed
                    }
                    scope {
                        imageScope {
                            registry
                            remote
                            tag
                        }
                    }
                    cves {
                        cves
                    }
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
    expiresOn?: string | number;
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
