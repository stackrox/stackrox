import { SlimUser } from './user.proto';

export type RequestComment = {
    id: string;
    user: SlimUser;
    message: string;
    createdAt: string;
};

export type RequestState = 'PENDING' | 'APPROVED' | 'DENIED';

export type DeferralRequest = {
    expiresOn?: string;
    expiresWhenFixed?: boolean;
};

export type Scope = {
    imageScope?: {
        name: string;
        tagRegex: string;
    };
};
