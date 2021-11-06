/*
 * Replace TokenMetadata from backend proto with ApiToken type name, because clearer in frontend code.
 */
export type ApiToken = {
    id: string;
    name: string;
    roles: string[];
    issuedAt: string; // ISO 8601 date string
    expiration: string; // ISO 8601 date string
    revoked: boolean;
};
