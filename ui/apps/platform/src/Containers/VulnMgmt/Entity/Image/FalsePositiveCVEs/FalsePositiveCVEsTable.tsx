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

import { VulnerabilitySeverity } from 'messages/common';

import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import CVSSScoreLabel from 'Components/PatternFly/CVSSScoreLabel';
import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import useTableSelection from 'hooks/useTableSelection';
import { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { ComponentWhereCVEOccurs, VulnerabilityComment } from '../types';
import AffectedComponentsButton from '../AffectedComponents/AffectedComponentsButton';
import VulnerabilityCommentsButton from '../VulnerabilityComments/VulnerabilityCommentsButton';
import ReobserveCVEModal from './ReobserveCVEModal';

export type FalsePositiveCVERow = {
    id: string;
    cve: string;
    cvssScore: string;
    severity: VulnerabilitySeverity;
    components: ComponentWhereCVEOccurs[];
    comments: VulnerabilityComment[];
    applyTo: string;
};

export type FalsePositiveCVEsTableProps = {
    rows: FalsePositiveCVERow[];
};

function FalsePositiveCVEsTable({ rows }: FalsePositiveCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection<FalsePositiveCVERow>(rows);
    const [falsePositiveCVEsToBeReobserved, setFalsePositiveCVEsToBeReobserved] = useState<
        string[]
    >([]);

    function setSelectedCVEFalsePositivesToBeCancelled() {
        const selectedIds = getSelectedIds();
        setFalsePositiveCVEsToBeReobserved(selectedIds);
    }

    function cancelReobserveCVE() {
        setFalsePositiveCVEsToBeReobserved([]);
    }

    function completeReobserveCVE() {
        onClearAll();
        setFalsePositiveCVEsToBeReobserved([]);
    }

    function requestReobserveCVE(values) {
        const promise = new Promise<FormResponseMessage>((resolve, reject) => {
            setTimeout(() => {
                if (values?.comment === 'blah') {
                    const formMessage = {
                        message: 'Successfully reobserved CVE',
                        isError: false,
                    };
                    resolve(formMessage);
                } else {
                    const formMessage = { message: 'API is not hooked up yet', isError: true };
                    reject(formMessage);
                }
            }, 2000);
        });
        return promise;
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
                                onClick={setSelectedCVEFalsePositivesToBeCancelled}
                            >
                                Reobserve CVE ({numSelected})
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
                        <Th>Severity</Th>
                        <Th>CVSS score</Th>
                        <Th>Affected Components</Th>
                        <Th>Apply to</Th>
                        <Th>Comments</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => {
                        const actions = [
                            {
                                title: 'Reobserve CVE',
                                onClick: (event) => {
                                    event.preventDefault();
                                    setFalsePositiveCVEsToBeReobserved([row.id]);
                                },
                            },
                        ];

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
                                <Td dataLabel="CVSS score">
                                    <CVSSScoreLabel cvss={row.cvssScore} />
                                </Td>
                                <Td dataLabel="Affected components">
                                    <AffectedComponentsButton components={row.components} />
                                </Td>
                                <Td dataLabel="Apply to">{row.applyTo}</Td>
                                <Td dataLabel="Comments">
                                    <VulnerabilityCommentsButton
                                        cve={row.cve}
                                        comments={row.comments}
                                    />
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
            <ReobserveCVEModal
                isOpen={falsePositiveCVEsToBeReobserved.length !== 0}
                onSendRequest={requestReobserveCVE}
                onCompleteRequest={completeReobserveCVE}
                onCancel={cancelReobserveCVE}
            />
        </>
    );
}

export default FalsePositiveCVEsTable;
