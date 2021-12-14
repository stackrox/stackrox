import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';
import { useQuery, useApolloClient } from '@apollo/client';

import usePagination from 'hooks/patternfly/usePagination';
import {
    GetVulnerabilityRequestsData,
    GetVulnerabilityRequestsVars,
    GET_VULNERABILITY_REQUESTS,
} from '../vulnerabilityRequests.graphql';

import PendingApprovalsTable from './PendingApprovalsTable';

function PendingApprovals(): ReactElement {
    const client = useApolloClient();
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { loading: isLoading, data } = useQuery<
        GetVulnerabilityRequestsData,
        GetVulnerabilityRequestsVars
    >(GET_VULNERABILITY_REQUESTS, {
        variables: {
            query: 'Request Status:PENDING,APPROVED_PENDING_UPDATE+Expired Request:false',
            pagination: {
                limit: perPage,
                offset: (page - 1) * perPage,
                sortOption: {
                    field: 'id',
                    reversed: false,
                },
            },
        },
        fetchPolicy: 'network-only',
    });

    async function updateTable() {
        await client.refetchQueries({
            include: [GET_VULNERABILITY_REQUESTS],
        });
    }

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner size="sm" />
            </Bullseye>
        );
    }

    const rows = data?.results || [];

    return (
        <PendingApprovalsTable
            rows={rows}
            updateTable={updateTable}
            isLoading={isLoading}
            // @TODO: When backend puts "vulnerabilityRequestsCount" into GraphQL, use that
            itemCount={rows.length}
            page={page}
            perPage={perPage}
            onSetPage={onSetPage}
            onPerPageSelect={onPerPageSelect}
        />
    );
}

export default PendingApprovals;
