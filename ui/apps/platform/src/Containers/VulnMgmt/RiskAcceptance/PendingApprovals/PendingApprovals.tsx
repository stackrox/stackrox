import React, { ReactElement } from 'react';
import { PageSection, PageSectionVariants } from '@patternfly/react-core';

import usePagination from 'hooks/patternfly/usePagination';
import queryService from 'utils/queryService';
import { getHasSearchApplied } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import PendingApprovalsTable from './PendingApprovalsTable';
import useVulnerabilityRequests from '../useVulnerabilityRequests';

function setDefaultSearchFields(searchFilter: SearchFilter): SearchFilter {
    let modifiedSearchObject = { ...searchFilter };
    if (!modifiedSearchObject['Request Status']) {
        modifiedSearchObject['Request Status'] = ['Pending', 'APPROVED_PENDING_UPDATE'];
    }
    modifiedSearchObject = { ...modifiedSearchObject, 'Expired Request': 'false' };
    return modifiedSearchObject;
}

function PendingApprovals(): ReactElement {
    const { searchFilter, setSearchFilter } = useURLSearch();

    const modifiedSearchObject = setDefaultSearchFields(searchFilter);
    /*
     * Due to backend limitations with the inability to index on the Request ID,
     * we must pass the search query for "Request ID" using
     * a separate GraphQL variable
     */
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
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
                <EmptyStateTemplate title="No pending requests to approve." headingLevel="h2" />
            </PageSection>
        );
    }

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
