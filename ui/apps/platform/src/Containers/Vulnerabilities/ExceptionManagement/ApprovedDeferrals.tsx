import React, { useCallback } from 'react';
import {
    Bullseye,
    Button,
    PageSection,
    Pagination,
    Spinner,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { FileAltIcon, SearchIcon } from '@patternfly/react-icons';

import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import useRestQuery from 'hooks/useRestQuery';
import { fetchVulnerabilityExceptions } from 'services/VulnerabilityExceptionService';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import PageTitle from 'Components/PageTitle';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
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

import {
    SearchOption,
    REQUEST_NAME_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    REQUESTER_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
} from '../searchOptions';
import { getTableUIState } from '../../../utils/getTableUIState';
import { DEFAULT_VM_PAGE_SIZE } from '../constants';

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

function ApprovedDeferrals() {
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
                    'Request Status': ['APPROVED', 'APPROVED_PENDING_UPDATE'],
                    'Requested Vulnerability State': 'DEFERRED',
                    'Expired Request': 'false',
                },
                sortOption,
                page - 1,
                perPage
            ),
        [searchFilter, sortOption, page, perPage]
    );
    // TODO: Consider changing the name of "loading" to "isLoading" - https://issues.redhat.com/browse/ROX-22865
    const { data, loading: isLoading, error } = useRestQuery(vulnerabilityExceptionsFn);

    const tableUIState = getTableUIState({
        isLoading,
        data,
        error,
        searchFilter,
    });
    function onFilterChange() {
        setPage(1);
    }

    if (tableUIState.type === 'ERROR') {
        return (
            <PageSection variant="light">
                <TableErrorComponent
                    error={tableUIState.error}
                    message="An error occurred. Try refreshing again"
                />
            </PageSection>
        );
    }

    return (
        <PageSection>
            <PageTitle title="Exception Management - Approved Deferrals" />
            <Toolbar>
                <ToolbarContent>
                    <FilterAutocompleteSelect
                        searchFilter={searchFilter}
                        onFilterChange={(newFilter) => setSearchFilter(newFilter)}
                        searchOptions={searchOptions}
                    />
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
                    <ToolbarGroup aria-label="applied search filters" className="pf-v5-u-w-100">
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
                <Tbody>
                    {tableUIState.type === 'LOADING' && (
                        <Tr>
                            <Td colSpan={7}>
                                <Bullseye>
                                    <Spinner aria-label="Loading table data" />
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {tableUIState.type === 'EMPTY' && (
                        <Tr>
                            <Td colSpan={7}>
                                <Bullseye>
                                    <EmptyStateTemplate
                                        title="No approved deferral requests"
                                        headingLevel="h2"
                                        icon={FileAltIcon}
                                    >
                                        <Text>
                                            There are currently no approved deferral requests. Feel
                                            free to review pending requests or return to your
                                            dashboard.
                                        </Text>
                                    </EmptyStateTemplate>
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {tableUIState.type === 'FILTERED_EMPTY' && (
                        <Tr>
                            <Td colSpan={7}>
                                <Bullseye>
                                    <EmptyStateTemplate
                                        title="No results found"
                                        headingLevel="h2"
                                        icon={SearchIcon}
                                    >
                                        <Text>
                                            We couldn’t find any items matching your search
                                            criteria. Try adjusting your filters or search terms for
                                            better results
                                        </Text>
                                        <Button
                                            variant="link"
                                            onClick={() => {
                                                setPage(1);
                                                setSearchFilter({});
                                            }}
                                        >
                                            Clear search filters
                                        </Button>
                                    </EmptyStateTemplate>
                                </Bullseye>
                            </Td>
                        </Tr>
                    )}
                    {(tableUIState.type === 'COMPLETE' || tableUIState.type === 'POLLING') &&
                        tableUIState.data.map((exception) => {
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
                        })}
                </Tbody>
            </Table>
        </PageSection>
    );
}

export default ApprovedDeferrals;
