import React, { ReactElement, useState } from 'react';

import usePagination from 'hooks/patternfly/usePagination';

import { SearchFilter } from 'types/search';
import queryService from 'utils/queryService';
import PendingApprovalsTable from './PendingApprovalsTable';
import useVulnerabilityRequests from '../useVulnerabilityRequests';

function PendingApprovals(): ReactElement {
    const [searchFilter, setSearchFilter] = useState<SearchFilter>({});
    let modifiedSearchObject = { ...searchFilter };
    if (!modifiedSearchObject['Request Status']) {
        modifiedSearchObject['Request Status'] = ['Pending', 'APPROVED_PENDING_UPDATE'];
    }
    modifiedSearchObject = { ...modifiedSearchObject, 'Expired Request': 'false' };
    const query = queryService.objectToWhereClause(modifiedSearchObject);

    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { isLoading, data, refetchQuery } = useVulnerabilityRequests({
        query,
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'id',
                reversed: false,
            },
        },
    });

    const rows = data?.vulnerabilityRequests || [];
    const itemCount = data?.vulnerabilityRequestsCount || 0;

    return (
        <PendingApprovalsTable
            rows={rows}
            updateTable={refetchQuery}
            isLoading={isLoading}
            itemCount={itemCount}
            searchFilter={searchFilter}
            setSearchFilter={setSearchFilter}
            page={page}
            perPage={perPage}
            onSetPage={onSetPage}
            onPerPageSelect={onPerPageSelect}
        />
    );
}

export default PendingApprovals;
