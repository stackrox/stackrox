/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React, { ReactElement, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
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
import { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import DeferralRequestModal from './DeferralRequestModal';

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
};

function ObservedCVEsTable({ rows }: ObservedCVEsTableProps): ReactElement {
    const { selected, allRowsSelected, numSelected, onSelect, onSelectAll, getSelectedIds } =
        useTableSelection<ObservedCVERow>(rows);
    const [cvesToBeDeferred, setCVEsToBeDeferred] = useState<string[]>([]);

    function setSelectedCVEsToBeDeferred() {
        const selectedIds = getSelectedIds();
        setCVEsToBeDeferred(selectedIds);
    }

    function cancelDeferringCVEs() {
        setCVEsToBeDeferred([]);
    }

    // @TODO: Convert the form values to the proper values used in the API for deferring a CVE
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    function completeDeferral(values) {
        // @TODO: call parent function that will send out an API call to mark as deferral
        const promise = new Promise<FormResponseMessage>((response, reject) => {
            setTimeout(() => {
                const formMessage = { message: 'API is not hooked up yet', isError: true };
                reject(formMessage);
            }, 2000);
        });
        return promise;
    }

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
                                onClick={setSelectedCVEsToBeDeferred}
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
                    {rows.map((row, rowIndex) => {
                        const actions = [
                            {
                                title: 'Defer CVE',
                                onClick: (event) => {
                                    event.preventDefault();
                                    setCVEsToBeDeferred([row.id]);
                                },
                            },
                            {
                                title: 'Mark as False Positive',
                                onClick: (event) => {
                                    event.preventDefault();
                                },
                            },
                            {
                                isSeparator: true,
                            },
                            {
                                title: 'Reject deferral',
                                onClick: (event) => {
                                    event.preventDefault();
                                },
                            },
                        ];
                        return (
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
                        );
                    })}
                </Tbody>
            </TableComposable>
            <DeferralRequestModal
                isOpen={cvesToBeDeferred.length !== 0}
                onCompleteDeferral={completeDeferral}
                onCancelDeferral={cancelDeferringCVEs}
            />
        </>
    );
}

export default ObservedCVEsTable;
