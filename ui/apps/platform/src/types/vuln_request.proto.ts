import { SlimUser } from './user.proto';

// TODO Revisit this file once ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL is removed to cleanup if the types are orphaned

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
