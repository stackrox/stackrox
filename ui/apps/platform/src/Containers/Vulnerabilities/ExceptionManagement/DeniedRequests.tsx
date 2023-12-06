import React, { useCallback } from 'react';
import {
    Bullseye,
    PageSection,
    Pagination,
    Spinner,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import useURLPagination from 'hooks/useURLPagination';

import useURLSearch from 'hooks/useURLSearch';
import useRestQuery from 'hooks/useRestQuery';
import useURLSort from 'hooks/useURLSort';
import { fetchVulnerabilityExceptions } from 'services/VulnerabilityExceptionService';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import NotFoundMessage from 'Components/NotFoundMessage';
import {
    RequestExpires,
    RequestIDLink,
    RequestedAction,
    RequestedItems,
    RequestCreatedAt,
    Requester,
    RequestScope,
} from './components/ExceptionRequestTableCells';
import FilterAutocompleteSelect from '../components/FilterAutocomplete';
import TableErrorComponent from '../WorkloadCves/components/TableErrorComponent';
import {
    SearchOption,
    REQUEST_NAME_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    REQUESTER_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
} from '../searchOptions';

const searchOptions: SearchOption[] = [
    REQUEST_NAME_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    REQUESTER_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
];

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
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);
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
                page - 1,
                perPage
            ),
        [searchFilter, sortOption, page, perPage]
    );
    const { data, loading, error } = useRestQuery(vulnerabilityExceptionsFn);

    function onFilterChange() {
        setPage(1);
    }

    if (loading && !data) {
        return (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <PageSection variant="light">
                <TableErrorComponent
                    error={error}
                    message="An error occurred. Try refreshing again"
                />
            </PageSection>
        );
    }

    if (!data) {
        return (
            <NotFoundMessage
                title="404: We couldn't find that page"
                message="Pending vulnerability exception requests could not be found."
            />
        );
    }

    // @TODO: Add a "No results" type of view when there is no data to show. Also, add them for the other tables.

    return (
        <PageSection>
            <Toolbar>
                <ToolbarContent>
                    <FilterAutocompleteSelect
                        searchFilter={searchFilter}
                        onFilterChange={(newFilter) => setSearchFilter(newFilter)}
                        searchOptions={searchOptions}
                    />
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
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
                    <ToolbarGroup aria-label="applied search filters" className="pf-u-w-100">
                        <SearchFilterChips
                            onFilterChange={onFilterChange}
                            filterChipGroupDescriptors={searchOptions.map(({ label, value }) => {
                                return {
                                    displayName: label,
                                    searchFilterName: value,
                                };
                            })}
                        />
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>
            <TableComposable borders={false}>
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
                <Tbody>
                    {data.map((exception) => {
                        const { id, name, requester, createdAt, scope } = exception;
                        return (
                            <Tr key={id}>
                                <Td>
                                    <RequestIDLink id={id} name={name} context="CURRENT" />
                                </Td>
                                <Td>
                                    <Requester requester={requester} />
                                </Td>
                                <Td>
                                    <RequestedAction exception={exception} context="CURRENT" />
                                </Td>
                                <Td>
                                    <RequestCreatedAt createdAt={createdAt} />
                                </Td>
                                <Td>
                                    <RequestExpires exception={exception} context="CURRENT" />
                                </Td>
                                <Td>
                                    <RequestScope scope={scope} />
                                </Td>
                                <Td>
                                    <RequestedItems exception={exception} context="CURRENT" />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
        </PageSection>
    );
}

export default DeniedRequests;
