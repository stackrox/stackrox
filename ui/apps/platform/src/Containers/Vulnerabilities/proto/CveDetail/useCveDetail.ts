import { gql, useQuery } from '@apollo/client';

export type ProtoAdvisory = {
    id: string;
    advisoryId: string;
    cveName: string;
    severity: string;
    cvss: number;
    source: string;
    fixable: boolean;
    fixedBy: string;
    description: string;
    publishedDate: string;
};

export const PROTO_CVE_DETAIL = gql`
    query protoCVEDetail($cveName: String!) {
        protoCVEDetail(cveName: $cveName) {
            id
            advisoryId
            cveName
            severity
            cvss
            source
            fixable
            fixedBy
            description
            publishedDate
        }
    }
`;

type ProtoCVEDetailData = {
    protoCVEDetail: ProtoAdvisory[];
};

type ProtoCVEDetailVars = {
    cveName: string;
};

/**
 * Fetches prototype CVE detail (advisories) from the GraphQL API.
 */
export function useCveDetail(cveName: string) {
    return useQuery<ProtoCVEDetailData, ProtoCVEDetailVars>(PROTO_CVE_DETAIL, {
        variables: { cveName },
        skip: !cveName,
        fetchPolicy: 'cache-and-network',
    });
}
