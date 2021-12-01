import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';
import { useQuery, useApolloClient } from '@apollo/client';

import {
    GetPendingApprovalsData,
    GetPendingApprovalsVars,
    GET_PENDING_APPROVALS,
} from './pendingApprovals.graphql';

import PendingApprovalsTable from './PendingApprovalsTable';

function PendingApprovals(): ReactElement {
    const client = useApolloClient();
    const { loading: isLoading, data } = useQuery<GetPendingApprovalsData, GetPendingApprovalsVars>(
        GET_PENDING_APPROVALS,
        {
            variables: {
                query: 'Request Status:PENDING',
                pagination: {
                    limit: 20,
                    offset: 0,
                    sortOption: {
                        field: 'id',
                        reversed: false,
                    },
                },
            },
        }
    );

    async function updateTable() {
        await client.refetchQueries({
            include: [GET_PENDING_APPROVALS],
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

    return <PendingApprovalsTable rows={rows} updateTable={updateTable} isLoading={isLoading} />;
}

export default PendingApprovals;
