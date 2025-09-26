import { gql, useQuery } from '@apollo/client';

import {
    ResourceCountByCveSeverityAndStatus,
    resourceCountByCveSeverityAndStatusFragment,
} from '../../components/CvesByStatusSummaryCard';

const nodeSummaryDataQuery = gql`
    ${resourceCountByCveSeverityAndStatusFragment}
    query getNodeVulnSummary($id: ID!, $query: String!) {
        node(id: $id) {
            id
            nodeCVECountBySeverity(query: $query) {
                ...ResourceCountsByCVESeverityAndStatus
            }
        }
    }
`;

export default function useNodeSummaryData(id: string, query: string) {
    return useQuery<
        {
            node: {
                id: string;
                nodeCVECountBySeverity: ResourceCountByCveSeverityAndStatus;
            };
        },
        {
            id: string;
            query: string;
        }
    >(nodeSummaryDataQuery, {
        variables: {
            id,
            query,
        },
    });
}
