/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td, IActions } from '@patternfly/react-table';
import {
    Button,
    ButtonVariant,
    Divider,
    DropdownItem,
    InputGroup,
    TextInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import useTableSelection from 'hooks/useTableSelection';
import { VulnerabilitySeverity } from 'messages/common';

import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import CVSSScoreLabel from 'Components/PatternFly/CVSSScoreLabel';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';

export type ObservedCVERow = {
    id: string;
    cve: string;
    isFixable: boolean;
    severity: VulnerabilitySeverity;
    cvssScore: string;
    components: { name: string }[];
    discoveredAt: string;
};

export type ObservedCVEsTableProps = {
    rows: ObservedCVERow[];
    actions: IActions;
};

function ObservedCVEsTable({ rows, actions }: ObservedCVEsTableProps): ReactElement {
    const { selected, allRowsSelected, numSelected, onSelect, onSelectAll } =
        useTableSelection<ObservedCVERow>(rows);

    function deferSelectedCVEs() {}

    function markFalsePositiveSelectedCVEs() {}

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
                                key="upgrade"
                                component="button"
                                onClick={deferSelectedCVEs}
                            >
                                Defer CVE ({numSelected})
                            </DropdownItem>
                            <DropdownItem
                                key="delete"
                                component="button"
                                onClick={markFalsePositiveSelectedCVEs}
                            >
                                Mark false positive ({numSelected})
                            </DropdownItem>
                        </BulkActionsDropdown>
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
                        <Th>Fixable</Th>
                        <Th>Severity</Th>
                        <Th>CVSS score</Th>
                        <Th>Affected components</Th>
                        <Th>Discovered</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => (
                        <Tr key={rowIndex}>
                            <Td
                                select={{
                                    rowIndex,
                                    onSelect,
                                    isSelected: selected[rowIndex],
                                }}
                            />
                            <Td dataLabel="Cell">{row.cve}</Td>
                            <Td dataLabel="Fixable">{row.isFixable ? 'Yes' : 'No'}</Td>
                            <Td dataLabel="Severity">
                                <VulnerabilitySeverityLabel severity={row.severity} />
                            </Td>
                            <Td dataLabel="CVSS score">
                                <CVSSScoreLabel cvss={row.cvssScore} />
                            </Td>
                            <Td dataLabel="Affected components">{row.components.length}</Td>
                            <Td dataLabel="Discovered">{row.discoveredAt}</Td>
                            <Td
                                className="pf-u-text-align-right"
                                actions={{
                                    items: actions,
                                }}
                            />
                        </Tr>
                    ))}
                </Tbody>
            </TableComposable>
        </>
    );
}

export default ObservedCVEsTable;
