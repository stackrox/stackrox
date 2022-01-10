import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import {
    Button,
    ButtonVariant,
    Divider,
    DropdownItem,
    InputGroup,
    Pagination,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import { UsePaginationResult } from 'hooks/patternfly/usePagination';
import usePermissions from 'hooks/usePermissions';
import AffectedComponentsButton from '../AffectedComponents/AffectedComponentsButton';
import { VulnerabilityWithRequest } from '../imageVulnerabilities.graphql';
import { FalsePositiveCVEsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import FalsePositiveCVEActionsColumn from './FalsePositiveCVEActionsColumns';
import RequestCommentsButton from '../RequestComments/RequestCommentsButton';
import VulnerabilityRequestScope from '../PendingApprovals/VulnerabilityRequestScope';

export type FalsePositiveCVEsTableProps = {
    rows: VulnerabilityWithRequest[];
    isLoading: boolean;
    itemCount: number;
    updateTable: () => void;
} & UsePaginationResult;

function FalsePositiveCVEsTable({
    rows,
    itemCount,
    page,
    perPage,
    onSetPage,
    onPerPageSelect,
    updateTable,
}: FalsePositiveCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection<VulnerabilityWithRequest>(rows);
    const [vulnsToBeAssessed, setVulnsToBeAssessed] = useState<FalsePositiveCVEsToBeAssessed>(null);
    const { undoVulnRequests } = useRiskAcceptance({
        requestIDs: vulnsToBeAssessed?.requestIDs || [],
    });
    const { currentUserName, hasReadWriteAccess } = usePermissions();

    function cancelAssessment() {
        setVulnsToBeAssessed(null);
    }

    async function completeAssessment() {
        onClearAll();
        setVulnsToBeAssessed(null);
        updateTable();
    }

    const canApproveRequests = hasReadWriteAccess('VulnerabilityManagementApprovals');
    const canCreateRequests = hasReadWriteAccess('VulnerabilityManagementRequests');

    const selectedIds = getSelectedIds();
    const selectedFalsePositivesToReobserve = rows
        .filter((row) => {
            return (
                selectedIds.includes(row.id) &&
                (canApproveRequests ||
                    (canCreateRequests &&
                        row.vulnerabilityRequest.requestor.name === currentUserName))
            );
        })
        .map((row) => {
            return row.vulnerabilityRequest.id;
        });

    return (
        <>
            <Toolbar id="toolbar">
                <ToolbarContent>
                    <ToolbarItem>
                        {/* @TODO: This is just a place holder. Put the correct search filter here */}
                        <InputGroup>
                            <TextInput
                                name="textInput1"
                                id="textInput1"
                                type="search"
                                aria-label="search input example"
                            />
                            <Button
                                variant={ButtonVariant.control}
                                aria-label="search button for search input"
                            >
                                <SearchIcon />
                            </Button>
                        </InputGroup>
                    </ToolbarItem>
                    <ToolbarItem variant="separator" />
                    <ToolbarItem>
                        <BulkActionsDropdown isDisabled={numSelected === 0}>
                            <DropdownItem
                                key="undo false positives"
                                component="button"
                                onClick={() =>
                                    setVulnsToBeAssessed({
                                        type: 'FALSE_POSITIVE',
                                        action: 'UNDO',
                                        requestIDs: selectedFalsePositivesToReobserve,
                                    })
                                }
                                isDisabled={selectedFalsePositivesToReobserve.length === 0}
                            >
                                Reobserve CVEs ({selectedFalsePositivesToReobserve.length})
                            </DropdownItem>
                        </BulkActionsDropdown>
                    </ToolbarItem>
                    <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                        <Pagination
                            itemCount={itemCount}
                            page={page}
                            onSetPage={onSetPage}
                            perPage={perPage}
                            onPerPageSelect={onPerPageSelect}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            <TableComposable aria-label="Observed CVEs Table" variant="compact" borders>
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                onSelect: onSelectAll,
                                isSelected: allRowsSelected,
                            }}
                        />
                        <Th>CVE</Th>
                        <Th>Severity</Th>
                        <Th>Affected Components</Th>
                        <Th>Comments</Th>
                        <Th>Apply to</Th>
                        <Th>Approver</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => {
                        const canReobserveCVE =
                            canApproveRequests ||
                            (canCreateRequests &&
                                row.vulnerabilityRequest.requestor.name === currentUserName);

                        return (
                            <Tr key={row.cve}>
                                <Td
                                    select={{
                                        rowIndex,
                                        onSelect,
                                        isSelected: selected[rowIndex],
                                    }}
                                />
                                <Td dataLabel="Cell">{row.cve}</Td>
                                <Td dataLabel="Severity">
                                    <VulnerabilitySeverityLabel severity={row.severity} />
                                </Td>
                                <Td dataLabel="Affected components">
                                    <AffectedComponentsButton components={row.components} />
                                </Td>
                                <Td dataLabel="Comments">
                                    <RequestCommentsButton
                                        comments={row.vulnerabilityRequest.comments}
                                        cve={row.vulnerabilityRequest.cves.ids[0]}
                                    />
                                </Td>
                                <Td dataLabel="Apply to">
                                    <VulnerabilityRequestScope
                                        scope={row.vulnerabilityRequest.scope}
                                    />
                                </Td>
                                <Td dataLabel="Approver">
                                    {row.vulnerabilityRequest.approvers
                                        .map((user) => user.name)
                                        .join(',')}
                                </Td>
                                <Td className="pf-u-text-align-right">
                                    <FalsePositiveCVEActionsColumn
                                        row={row}
                                        setVulnsToBeAssessed={setVulnsToBeAssessed}
                                        canReobserveCVE={canReobserveCVE}
                                    />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </TableComposable>
            <UndoVulnRequestModal
                type="FALSE_POSITIVE"
                isOpen={vulnsToBeAssessed?.action === 'UNDO'}
                numRequestsToBeAssessed={vulnsToBeAssessed?.requestIDs.length || 0}
                onSendRequest={undoVulnRequests}
                onCompleteRequest={completeAssessment}
                onCancel={cancelAssessment}
            />
        </>
    );
}

export default FalsePositiveCVEsTable;
