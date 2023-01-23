import React, { ReactElement } from 'react';
import { PageSection, PageSectionVariants } from '@patternfly/react-core';

import usePagination from 'hooks/patternfly/usePagination';
import queryService from 'utils/queryService';
import { getHasSearchApplied } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useVulnerabilityRequests from '../useVulnerabilityRequests';
import ApprovedDeferralsTable from './ApprovedDeferralsTable';

function ApprovedDeferrals(): ReactElement {
    const { searchFilter, setSearchFilter } = useURLSearch();

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
    const hasSearchApplied = getHasSearchApplied(searchFilter);

    if (!isLoading && rows && rows.length === 0 && !hasSearchApplied) {
        return (
            <PageSection variant={PageSectionVariants.light} isFilled>
                <EmptyStateTemplate title="No deferral requests were approved." headingLevel="h2" />
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
