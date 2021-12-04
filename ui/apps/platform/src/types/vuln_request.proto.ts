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
