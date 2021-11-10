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
import FalsePositiveRequestModal from './FalsePositiveRequestModal';
import ComponentsModal from './ComponentsModal';
import { ComponentWhereCVEOccurs } from '../types';

export type CVEsToBeAssessed = {
    type: 'DEFERRAL' | 'FALSE_POSITIVE';
    ids: string[];
} | null;

export type ObservedCVERow = {
    id: string;
    cve: string;
    isFixable: boolean;
    severity: VulnerabilitySeverity;
    cvssScore: string;
    components: ComponentWhereCVEOccurs[];
    discoveredAt: string;
};

export type ObservedCVEsTableProps = {
    rows: ObservedCVERow[];
};

function ObservedCVEsTable({ rows }: ObservedCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        getSelectedIds,
        onClearAll,
    } = useTableSelection<ObservedCVERow>(rows);
    const [cvesToBeAssessed, setCVEsToBeAssessed] = useState<CVEsToBeAssessed>(null);
    const [selectedComponents, setSelectedComponents] = useState<ComponentWhereCVEOccurs[]>([]);

    function setSelectedCVEsToBeDeferred() {
        const selectedIds = getSelectedIds();
        setCVEsToBeAssessed({ type: 'DEFERRAL', ids: selectedIds });
    }

    function setSelectedCVEsToBeMarkedFalsePositive() {
        const selectedIds = getSelectedIds();
        setCVEsToBeAssessed({ type: 'FALSE_POSITIVE', ids: selectedIds });
    }

    function cancelAssessment() {
        setCVEsToBeAssessed(null);
    }

    function completeAssessment() {
        onClearAll();
        setCVEsToBeAssessed(null);
    }

    // @TODO: Convert the form values to the proper values used in the API for deferring a CVE
    function requestDeferral(values) {
        // @TODO: call parent function that will send out an API call to mark as deferral
        const promise = new Promise<FormResponseMessage>((resolve, reject) => {
            setTimeout(() => {
                if (values?.comment === 'blah') {
                    const formMessage = { message: 'Successfully deferred CVE', isError: false };
                    resolve(formMessage);
                } else {
                    const formMessage = { message: 'API is not hooked up yet', isError: true };
                    reject(formMessage);
                }
            }, 2000);
        });
        return promise;
    }

    // @TODO: Convert the form values to the proper values used in the API for marking a CVE as
    // false positive
    function requestFalsePositive(values) {
        // @TODO: call parent function that will send out an API call to mark as deferral
        const promise = new Promise<FormResponseMessage>((resolve, reject) => {
            if (values?.comment === 'blah') {
                const formMessage = { message: 'Successfully deferred CVE', isError: false };
                resolve(formMessage);
            } else {
                const formMessage = { message: 'API is not hooked up yet', isError: true };
                reject(formMessage);
            }
        });
        return promise;
    }

    function onCloseComponentsModal() {
        setSelectedComponents([]);
    }

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
                                onClick={setSelectedCVEsToBeMarkedFalsePositive}
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
                                    setCVEsToBeAssessed({ type: 'DEFERRAL', ids: [row.id] });
                                },
                            },
                            {
                                title: 'Mark as False Positive',
                                onClick: (event) => {
                                    event.preventDefault();
                                    setCVEsToBeAssessed({ type: 'DEFERRAL', ids: [row.id] });
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

                        function showComponents() {
                            setSelectedComponents(row.components);
                        }

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
                                <Td dataLabel="Affected components">
                                    <Button variant="link" isInline onClick={showComponents}>
                                        {row.components.length} components
                                    </Button>
                                </Td>
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
                isOpen={cvesToBeAssessed?.type === 'DEFERRAL' && cvesToBeAssessed?.ids.length !== 0}
                onRequestDeferral={requestDeferral}
                onCompleteDeferral={completeAssessment}
                onCancelDeferral={cancelAssessment}
            />
            <FalsePositiveRequestModal
                isOpen={
                    cvesToBeAssessed?.type === 'FALSE_POSITIVE' &&
                    cvesToBeAssessed?.ids.length !== 0
                }
                onRequestFalsePositive={requestFalsePositive}
                onCompleteFalsePositive={completeAssessment}
                onCancelFalsePositive={cancelAssessment}
            />
            <ComponentsModal components={selectedComponents} onClose={onCloseComponentsModal} />
        </>
    );
}

export default ObservedCVEsTable;
