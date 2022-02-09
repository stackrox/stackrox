import React, { ReactElement } from 'react';

import usePagination from 'hooks/patternfly/usePagination';
import queryService from 'utils/queryService';
import useSearch from 'hooks/useSearch';
import { PageSection, PageSectionVariants } from '@patternfly/react-core';
import ACSEmptyState from 'Components/ACSEmptyState';
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
    const requestID = modifiedSearchObject['Request ID'];
    delete modifiedSearchObject['Request ID'];
    const query = queryService.objectToWhereClause(modifiedSearchObject);

    const { page, perPage, onSetPage, onPerPageSelect } = usePagination();
    const { isLoading, data, refetchQuery } = useVulnerabilityRequests({
        query,
        requestID,
        pagination: {
            limit: perPage,
            offset: (page - 1) * perPage,
            sortOption: {
                field: 'Last Updated',
                reversed: false,
            },
        },
    });

    const rows = data?.vulnerabilityRequests || [];
    const itemCount = data?.vulnerabilityRequestsCount || 0;

    if (!isLoading && rows && rows.length === 0) {
        return (
            <PageSection variant={PageSectionVariants.light} isFilled>
                <ACSEmptyState title="No deferral requests were approved." />
            </PageSection>
        );
    }

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
