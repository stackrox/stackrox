import React from 'react';
import {
    PageSection,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import useURLPagination from 'hooks/useURLPagination';

import useURLSearch from 'hooks/useURLSearch';
import {
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    REQUESTER_SEARCH_OPTION,
    REQUEST_NAME_SEARCH_OPTION,
    SearchOption,
} from 'Containers/Vulnerabilities/components/SearchOptionsDropdown';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
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
import { approvedDeferrals as vulnerabilityExceptions } from './mockUtils';

const searchOptions: SearchOption[] = [
    REQUEST_NAME_SEARCH_OPTION,
    IMAGE_CVE_SEARCH_OPTION,
    REQUESTER_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
];

function ApprovedDeferrals() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);

    function onFilterChange() {
        setPage(1);
    }

    return (
        <PageSection>
            <Toolbar>
                <ToolbarContent>
                    <FilterAutocompleteSelect
                        searchFilter={searchFilter}
                        setSearchFilter={setSearchFilter}
                        searchOptions={searchOptions}
                    />
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={1}
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
                        <Th>Request ID</Th>
                        <Th>Requester</Th>
                        <Th>Requested action</Th>
                        <Th>Requested</Th>
                        <Th>Expires</Th>
                        <Th>Scope</Th>
                        <Th>Requested items</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {vulnerabilityExceptions.map((exception) => {
                        const { id, name, requester, createdAt, scope } = exception;
                        return (
                            <Tr key={id}>
                                <Td>
                                    <RequestIDLink id={id} name={name} />
                                </Td>
                                <Td>
                                    <Requester requester={requester} />
                                </Td>
                                <Td>
                                    <RequestedAction
                                        exception={exception}
                                        context="APPROVED_DEFERRALS"
                                    />
                                </Td>
                                <Td>
                                    <RequestCreatedAt createdAt={createdAt} />
                                </Td>
                                <Td>
                                    <RequestExpires
                                        exception={exception}
                                        context="APPROVED_DEFERRALS"
                                    />
                                </Td>
                                <Td>
                                    <RequestScope scope={scope} />
                                </Td>
                                <Td>
                                    <RequestedItems
                                        exception={exception}
                                        context="APPROVED_DEFERRALS"
                                    />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
        </PageSection>
    );
}

export default ApprovedDeferrals;
