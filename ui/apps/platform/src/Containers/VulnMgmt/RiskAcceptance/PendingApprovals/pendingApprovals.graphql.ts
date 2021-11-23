import { gql } from '@apollo/client';

import { VulnerabilityState } from 'types/cve.proto';
import { SlimUser } from 'types/user.proto';
import { DeferralRequest, RequestComment, RequestState, Scope } from 'types/vuln_request.proto';

export type VulnerabilityRequest = {
    id: string;
    targetState: VulnerabilityState;
    status: RequestState;
    requestor: SlimUser;
    comments: RequestComment[];
    scope: Scope;
    deferralReq: DeferralRequest;
    cves: {
        ids: string[];
    };
};

export type GetPendingApprovalsData = {
    results: VulnerabilityRequest[];
};

export type GetPendingApprovalsVars = {
    query: string;
    pagination: {
        limit: number;
        offset: number;
        sortOption: {
            field: string;
            reversed: boolean;
        };
    };
};

// @TODO: We can create fragments for reusable pieces
export const GET_PENDING_APPROVALS = gql`
    query getPendingApprovals($query: String, $pagination: Pagination) {
        results: vulnerabilityRequests(query: $query, pagination: $pagination) {
            id
            targetState
            status
            requestor {
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
            scope {
                imageScope {
                    name
                    tagRegex
                }
            }
            deferralReq {
                expiresOn
                expiresWhenFixed
            }
            cves {
                ids
            }
        }
    }
`;
