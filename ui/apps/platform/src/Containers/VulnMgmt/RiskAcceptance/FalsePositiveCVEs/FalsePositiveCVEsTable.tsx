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
import AffectedComponentsButton from '../AffectedComponents/AffectedComponentsButton';
import { Vulnerability } from '../imageVulnerabilities.graphql';
import { FalsePositiveCVEsToBeAssessed } from './types';
import useRiskAcceptance from '../useRiskAcceptance';
import UndoVulnRequestModal from '../UndoVulnRequestModal';
import FalsePositiveCVEActionsColumn from './FalsePositiveCVEActionsColumns';

export type FalsePositiveCVEsTableProps = {
    rows: Vulnerability[];
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
    } = useTableSelection<Vulnerability>(rows);
    const [vulnsToBeAssessed, setVulnsToBeAssessed] = useState<FalsePositiveCVEsToBeAssessed>(null);
    const { undoVulnRequests } = useRiskAcceptance({
        requestIDs: vulnsToBeAssessed?.requestIDs || [],
    });

    function cancelAssessment() {
        setVulnsToBeAssessed(null);
    }

    async function completeAssessment() {
        onClearAll();
        setVulnsToBeAssessed(null);
        updateTable();
    }

    const selectedIds = getSelectedIds();
    const vulnRequestIds = rows
        .filter((row) => selectedIds.includes(row.id))
        .map((row) => {
            // @TODO: Once backend adds resolver for vulnRequests, access that and return the request id
            // This will fail when sending the API request for now
            return row.id;
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
                                        requestIDs: vulnRequestIds,
                                    })
                                }
                                isDisabled={vulnRequestIds.length === 0}
                            >
                                Reobserve CVEs ({vulnRequestIds.length})
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
                        <Th>Expiration</Th>
                        <Th>Apply to</Th>
                        <Th>Approver</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => {
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
                                <Td dataLabel="Comments">-</Td>
                                <Td dataLabel="Expiration">-</Td>
                                <Td dataLabel="Apply to">-</Td>
                                <Td dataLabel="Approver">-</Td>
                                <Td className="pf-u-text-align-right">
                                    <FalsePositiveCVEActionsColumn
                                        row={row}
                                        setVulnsToBeAssessed={setVulnsToBeAssessed}
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
