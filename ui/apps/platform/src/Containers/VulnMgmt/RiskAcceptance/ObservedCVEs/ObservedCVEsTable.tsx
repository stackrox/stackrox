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

import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import CVSSScoreLabel from 'Components/PatternFly/CVSSScoreLabel';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import DateTimeFormat from 'Components/PatternFly/DateTimeFormat';
import DeferralFormModal from './DeferralFormModal';
import FalsePositiveRequestModal from './FalsePositiveFormModal';
import { Vulnerability } from './observedCVEs.graphql';
import AffectedComponentsButton from '../AffectedComponents/AffectedComponentsButton';
import useDeferVulnerability from './useDeferVulnerability';
import useMarkFalsePositive from './useMarkFalsePositive';

export type CVEsToBeAssessed = {
    type: 'DEFERRAL' | 'FALSE_POSITIVE';
    ids: string[];
} | null;

export type ObservedCVERow = Vulnerability;

export type ObservedCVEsTableProps = {
    rows: ObservedCVERow[];
    isLoading: boolean;
    registry: string;
    remote: string;
    tag: string;
};

function ObservedCVEsTable({ rows, registry, remote, tag }: ObservedCVEsTableProps): ReactElement {
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
    const requestDeferral = useDeferVulnerability({
        cveIDs: cvesToBeAssessed?.ids || [],
        registry,
        remote,
        tag,
    });
    const requestFalsePositive = useMarkFalsePositive({
        cveIDs: cvesToBeAssessed?.ids || [],
        registry,
        remote,
        tag,
    });

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
                                    setCVEsToBeAssessed({ type: 'DEFERRAL', ids: [row.cve] });
                                },
                            },
                            {
                                title: 'Mark as False Positive',
                                onClick: (event) => {
                                    event.preventDefault();
                                    setCVEsToBeAssessed({ type: 'FALSE_POSITIVE', ids: [row.cve] });
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
                                    <CVSSScoreLabel
                                        cvss={row.cvss}
                                        scoreVersion={row.scoreVersion}
                                    />
                                </Td>
                                <Td dataLabel="Affected components">
                                    <AffectedComponentsButton components={row.components} />
                                </Td>
                                <Td dataLabel="Discovered">
                                    <DateTimeFormat time={row.discoveredAtImage} />
                                </Td>
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
            <DeferralFormModal
                isOpen={cvesToBeAssessed?.type === 'DEFERRAL'}
                numCVEsToBeAssessed={cvesToBeAssessed?.ids.length || 0}
                onSendRequest={requestDeferral}
                onCompleteRequest={completeAssessment}
                onCancelDeferral={cancelAssessment}
            />
            <FalsePositiveRequestModal
                isOpen={cvesToBeAssessed?.type === 'FALSE_POSITIVE'}
                numCVEsToBeAssessed={cvesToBeAssessed?.ids.length || 0}
                onSendRequest={requestFalsePositive}
                onCompleteRequest={completeAssessment}
                onCancelFalsePositive={cancelAssessment}
            />
        </>
    );
}

export default ObservedCVEsTable;
