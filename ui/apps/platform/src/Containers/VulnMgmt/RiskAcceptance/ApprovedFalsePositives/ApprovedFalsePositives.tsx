import React, { ReactElement, useState } from 'react';

import usePagination from 'hooks/patternfly/usePagination';
import queryService from 'utils/queryService';
import { SearchFilter } from 'types/search';
import useVulnerabilityRequests from '../useVulnerabilityRequests';
import ApprovedFalsePositivesTable from './ApprovedFalsePositivesTable';

function ApprovedFalsePositives(): ReactElement {
    const [searchFilter, setSearchFilter] = useState<SearchFilter>({});
    let modifiedSearchObject = { ...searchFilter };
    modifiedSearchObject = {
        ...modifiedSearchObject,
        'Expired Request': 'false',
        'Requested Vulnerability State': 'FALSE_POSITIVE',
        'Request Status': 'APPROVED',
    };
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
        <ApprovedFalsePositivesTable
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

export default ApprovedFalsePositives;
