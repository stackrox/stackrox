import { VulnerabilityState } from './cve.proto';
import { SlimUser } from './user.proto';

export type RequestComment = {
    id: string;
    user: SlimUser;
    message: string;
    createdAt: string;
};

export type RequestStatus = 'PENDING' | 'APPROVED' | 'DENIED' | 'APPROVED_PENDING_UPDATE';

export type DeferralRequest = {
    expiresOn?: string;
    expiresWhenFixed?: boolean;
};

export type Scope = {
    imageScope?: {
        registry: string;
        remote: string;
        tag: string;
    };
};

export type VulnerabilityRequest = {
    id: string;
    targetState: VulnerabilityState;
    status: RequestStatus;
    expired: boolean;
    requestor: SlimUser;
    approvers: SlimUser[];
    createdAt: string;
    lastUpdated: string;
    comments: RequestComment[];
    scope: Scope;
    deferralReq: {
        expiry: {
            expiresWhenFixed?: boolean;
            expiresOn?: string; // graphql.Time format (2021-12-03T18:25:39.397427643Z)
        };
    };
    cves: {
        ids: string[];
    };
};
