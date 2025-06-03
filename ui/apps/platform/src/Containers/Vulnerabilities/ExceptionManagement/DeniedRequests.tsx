import React, { useCallback } from 'react';
import { PageSection, Pagination, ToolbarItem } from '@patternfly/react-core';
import { Table, Td, Th, Thead, Tr } from '@patternfly/react-table';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useRestQuery from 'hooks/useRestQuery';
import useURLSort from 'hooks/useURLSort';
import { fetchVulnerabilityExceptions } from 'services/VulnerabilityExceptionService';

import PageTitle from 'Components/PageTitle';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { SearchFilter } from 'types/search';
import {
    RequestExpires,
    RequestIDLink,
    RequestedAction,
    RequestedItems,
    RequestCreatedAt,
    Requester,
    RequestScope,
} from './components/ExceptionRequestTableCells';
import { DEFAULT_VM_PAGE_SIZE } from '../constants';
import { getTableUIState } from '../../../utils/getTableUIState';
import AdvancedFiltersToolbar from '../components/AdvancedFiltersToolbar';
import {
    convertToFlatVulnRequestSearchFilterConfig, // vulnRequestSearchFilterConfig
} from './searchFilterConfig';

const sortFields = [
    'Request Name',
    'Requester User Name',
    'Created Time',
    'Request Expiry Time',
    'Image Registry Scope',
];
const defaultSortOption = {
    field: sortFields[0],
    direction: 'desc',
} as const;

function DeniedRequests() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const vulnRequestSearchFilterConfig = convertToFlatVulnRequestSearchFilterConfig(
        isFeatureFlagEnabled('ROX_FLATTEN_CVE_DATA')
    );

    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const vulnerabilityExceptionsFn = useCallback(
        () =>
            fetchVulnerabilityExceptions(
                {
                    ...searchFilter,
                    'Request Status': ['DENIED'],
                },
                sortOption,
                page,
                perPage
            ),
        [searchFilter, sortOption, page, perPage]
    );
    // TODO: Consider changing the name of "loading" to "isLoading" - https://issues.redhat.com/browse/ROX-22865
    const { data, isLoading, error } = useRestQuery(vulnerabilityExceptionsFn);

    const tableState = getTableUIState({
        isLoading,
        data,
        error,
        searchFilter,
    });

    function onFilterChange(searchFilter: SearchFilter) {
        setSearchFilter(searchFilter);
        setPage(1);
    }

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    if (tableState.type === 'ERROR') {
        return (
            <PageSection variant="light">
                <TableErrorComponent
                    error={tableState.error}
                    message="An error occurred. Try refreshing again"
                />
            </PageSection>
        );
    }

    return (
        <PageSection>
            <PageTitle title="Exception Management - Denied Requests" />
            <AdvancedFiltersToolbar
                searchFilterConfig={vulnRequestSearchFilterConfig}
                searchFilter={searchFilter}
                onFilterChange={onFilterChange}
                includeCveSeverityFilters={false}
                includeCveStatusFilters={false}
            >
                <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
                    <Pagination
                        toggleTemplate={({ firstIndex, lastIndex }) => (
                            <span>
                                <b>
                                    {firstIndex} - {lastIndex}
                                </b>{' '}
                                of <b>many</b>
                            </span>
                        )}
                        page={page}
                        perPage={perPage}
                        onSetPage={(_, newPage) => setPage(newPage)}
                        onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                        isCompact
                    />
                </ToolbarItem>
            </AdvancedFiltersToolbar>
            <Table borders={false}>
                <Thead noWrap>
                    <Tr>
                        <Th sort={getSortParams('Request Name')}>Request name</Th>
                        <Th sort={getSortParams('Requester User Name')}>Requester</Th>
                        <Th>Requested action</Th>
                        <Th sort={getSortParams('Created Time')}>Requested</Th>
                        <Th sort={getSortParams('Request Expiry Time')}>Expires</Th>
                        <Th sort={getSortParams('Image Registry Scope')}>Scope</Th>
                        <Th>Requested items</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={7}
                    emptyProps={{
                        title: 'No denied exception requests',
                        message:
                            'There are currently no denied exception requests. Feel free to review pending requests or return to your dashboard.',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) =>
                        data.map((exception) => {
                            const { id, name, requester, createdAt, scope } = exception;
                            return (
                                <Tr key={id}>
                                    <Td dataLabel="Request name">
                                        <RequestIDLink id={id} name={name} context="CURRENT" />
                                    </Td>
                                    <Td dataLabel="Requester">
                                        <Requester requester={requester} />
                                    </Td>
                                    <Td dataLabel="Requested action">
                                        <RequestedAction exception={exception} context="CURRENT" />
                                    </Td>
                                    <Td dataLabel="Requested">
                                        <RequestCreatedAt createdAt={createdAt} />
                                    </Td>
                                    <Td dataLabel="Expires">
                                        <RequestExpires exception={exception} context="CURRENT" />
                                    </Td>
                                    <Td dataLabel="Scope">
                                        <RequestScope scope={scope} />
                                    </Td>
                                    <Td dataLabel="Requested items">
                                        <RequestedItems exception={exception} context="CURRENT" />
                                    </Td>
                                </Tr>
                            );
                        })
                    }
                />
            </Table>
        </PageSection>
    );
}

export default DeniedRequests;
