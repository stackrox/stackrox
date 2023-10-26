import React from 'react';
import {
    PageSection,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import useURLPagination from 'hooks/useURLPagination';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';

import {
    ExpiresTableCell,
    RequestIDTableCell,
    RequestedActionTableCell,
    RequestedItemsTableCell,
    RequestedTableCell,
    RequesterTableCell,
    ScopeTableCell,
} from './components/ExceptionRequestTableCells';

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
        cves: ['CVE-2018-20839'],
    },
];

function PendingApprovals() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);

    return (
        <PageSection>
            <Toolbar>
                <ToolbarContent>
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
                        const { id, name, requester, createdAt, scope, cves } = exception;
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
                                        context="PENDING_REQUESTS"
                                    />
                                </Td>
                                <Td>
                                    <RequestedTableCell createdAt={createdAt} />
                                </Td>
                                <Td>
                                    <ExpiresTableCell
                                        exception={exception}
                                        context="PENDING_REQUESTS"
                                    />
                                </Td>
                                <Td>
                                    <ScopeTableCell scope={scope} />
                                </Td>
                                <Td>
                                    <RequestedItemsTableCell cves={cves} />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
        </PageSection>
    );
}

export default PendingApprovals;
