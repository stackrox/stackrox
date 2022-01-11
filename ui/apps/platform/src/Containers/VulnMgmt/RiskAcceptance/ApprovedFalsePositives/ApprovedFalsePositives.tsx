import React, { ReactElement } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import usePagination from 'hooks/patternfly/usePagination';
import useVulnerabilityRequests from '../useVulnerabilityRequests';
import ApprovedFalsePositivesTable from './ApprovedFalsePositivesTable';

function ApprovedFalsePositives(): ReactElement {
    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { isLoading, data, refetchQuery } = useVulnerabilityRequests({
        query: 'Request Status:APPROVED+Requested Vulnerability State:FALSE_POSITIVE+Expired Request:false',
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'id',
                reversed: false,
            },
        },
    });

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner isSVG size="sm" />
            </Bullseye>
        );
    }

    const rows = data?.results || [];

    return (
        <ApprovedFalsePositivesTable
            rows={rows}
            updateTable={refetchQuery}
            isLoading={isLoading} // @TODO: When backend puts "vulnerabilityRequestsCount" into GraphQL, use that
            itemCount={rows.length}
            page={page}
            perPage={perPage}
            onSetPage={onSetPage}
            onPerPageSelect={onPerPageSelect}
        />
    );
}

export default ApprovedFalsePositives;
