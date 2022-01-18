import React, { ReactElement } from 'react';

import usePagination from 'hooks/patternfly/usePagination';
import queryService from 'utils/queryService';
import useSearch from 'hooks/useSearch';
import useVulnerabilityRequests from '../useVulnerabilityRequests';
import ApprovedDeferralsTable from './ApprovedDeferralsTable';

function ApprovedDeferrals(): ReactElement {
    const { searchFilter, setSearchFilter } = useSearch();

    const modifiedSearchObject = {
        ...searchFilter,
        'Expired Request': 'false',
        'Requested Vulnerability State': 'DEFERRED',
    };
    if (!modifiedSearchObject['Request Status']) {
        modifiedSearchObject['Request Status'] = ['APPROVED'];
    }
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
        <ApprovedDeferralsTable
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

export default ApprovedDeferrals;
