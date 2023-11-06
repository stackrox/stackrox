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
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';

import useURLSearch from 'hooks/useURLSearch';
import {
    IMAGE_CVE_SEARCH_OPTION,
    IMAGE_SEARCH_OPTION,
    REQUESTER_SEARCH_OPTION,
    REQUEST_ID_SEARCH_OPTION,
    SearchOption,
} from 'Containers/Vulnerabilities/components/SearchOptionsDropdown';

import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import {
    ExpiresTableCell,
    RequestIDTableCell,
    RequestedActionTableCell,
    RequestedItemsTableCell,
    RequestedTableCell,
    RequesterTableCell,
    ScopeTableCell,
} from './components/ExceptionRequestTableCells';
import FilterAutocompleteSelect from '../components/FilterAutocomplete';

// @TODO: Use API data instead of hardcoded data
const vulnerabilityExceptions: VulnerabilityException[] = [
    {
        id: '4837bb34-5357-4b78-ad2b-188fc0b33e78',
        name: '4837bb34-5357-4b78-ad2b-188fc0b33e78',
        targetState: 'DEFERRED',
        exceptionStatus: 'APPROVED_PENDING_UPDATE',
        expired: false,
        requester: {
            id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
            name: 'admin',
        },
        createdAt: '2023-10-01T19:16:49.155480945Z',
        lastUpdated: '2023-10-01T19:16:49.155480945Z',
        comments: [
            {
                createdAt: '2023-10-23T19:16:49.155480945Z',
                id: 'c84b3f5f-4cad-4c4e-8a4a-97b821c2c373',
                message: 'asdf',
                user: {
                    id: 'sso:4df1b98c-24ed-4073-a9ad-356aec6bb62d:admin',
                    name: 'admin',
                },
            },
        ],
        scope: {
            imageScope: {
                registry: 'quay.io',
                remote: 'stackrox-io/scanner',
                tag: '.*',
            },
        },
        deferralRequest: {
            expiry: {
                expiryType: 'ALL_CVE_FIXABLE',
            },
        },
        deferralUpdate: {
            cves: ['CVE-2018-20839'],
            expiry: {
                expiryType: 'TIME',
                expiresOn: '2023-10-31T19:16:49.155480945Z',
            },
        },
        cves: ['CVE-2018-20839', 'CVE-2018-20840'],
    },
];

const searchOptions: SearchOption[] = [
    REQUEST_ID_SEARCH_OPTION,
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
                                    <RequestIDTableCell id={id} name={name} />
                                </Td>
                                <Td>
                                    <RequesterTableCell requester={requester} />
                                </Td>
                                <Td>
                                    <RequestedActionTableCell
                                        exception={exception}
                                        context="APPROVED_DEFERRALS"
                                    />
                                </Td>
                                <Td>
                                    <RequestedTableCell createdAt={createdAt} />
                                </Td>
                                <Td>
                                    <ExpiresTableCell
                                        exception={exception}
                                        context="APPROVED_DEFERRALS"
                                    />
                                </Td>
                                <Td>
                                    <ScopeTableCell scope={scope} />
                                </Td>
                                <Td>
                                    <RequestedItemsTableCell
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
