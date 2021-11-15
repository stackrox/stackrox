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
import useTableSelection from 'hooks/useTableSelection';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import VulnerabilitySeverityLabel from 'Components/PatternFly/VulnerabilitySeverityLabel';
import { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { ComponentWhereCVEOccurs, VulnerabilityComment } from '../types';
import AffectedComponentsButton from '../AffectedComponents/AffectedComponentsButton';
import CancelDeferralModal from './CancelDeferralModal';
import VulnerabilityCommentsButton from '../VulnerabilityComments/VulnerabilityCommentsButton';

export type DeferredCVERow = {
    id: string;
    cve: string;
    severity: VulnerabilitySeverity;
    components: ComponentWhereCVEOccurs[];
    comments: VulnerabilityComment[];
    expiresAt: string;
    applyTo: string;
    approver: string;
};

export type DeferredCVEsTableProps = {
    rows: DeferredCVERow[];
};

function DeferredCVEsTable({ rows }: DeferredCVEsTableProps): ReactElement {
    const {
        selected,
        allRowsSelected,
        numSelected,
        onSelect,
        onSelectAll,
        onClearAll,
        getSelectedIds,
    } = useTableSelection<DeferredCVERow>(rows);
    const [cveDeferralsToBeCancelled, setCVEDeferralsToBeCancelled] = useState<string[]>([]);

    function setSelectedCVEDeferralsToBeCancelled() {
        const selectedIds = getSelectedIds();
        setCVEDeferralsToBeCancelled(selectedIds);
    }

    function cancelCancellation() {
        setCVEDeferralsToBeCancelled([]);
    }

    function completeCancelDeferral() {
        onClearAll();
        setCVEDeferralsToBeCancelled([]);
    }

    // @TODO: Convert the form values to the proper values used in the API for cancelling a request
    function requestCancelDeferral(values) {
        // @TODO: call parent function that will send out an API call to cancel request
        const promise = new Promise<FormResponseMessage>((resolve, reject) => {
            setTimeout(() => {
                if (values?.comment === 'blah') {
                    const formMessage = {
                        message: 'Successfully cancelled request',
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
                                onClick={setSelectedCVEDeferralsToBeCancelled}
                            >
                                Cancel deferral ({numSelected})
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
                        <Th>Affected Components</Th>
                        <Th>Comments</Th>
                        <Th>Expiration</Th>
                        <Th>Apply to</Th>
                        <Th>Approver</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {rows.map((row, rowIndex) => {
                        const actions = [
                            {
                                title: 'Cancel deferral',
                                onClick: (event) => {
                                    event.preventDefault();
                                    setCVEDeferralsToBeCancelled([row.id]);
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
                                <Td dataLabel="Affected components">
                                    <AffectedComponentsButton components={row.components} />
                                </Td>
                                <Td dataLabel="Comments">
                                    <VulnerabilityCommentsButton
                                        cve={row.cve}
                                        comments={row.comments}
                                    />
                                </Td>
                                <Td dataLabel="Expiration">{row.expiresAt}</Td>
                                <Td dataLabel="Apply to">{row.applyTo}</Td>
                                <Td dataLabel="Approver">{row.approver}</Td>
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
            <CancelDeferralModal
                isOpen={cveDeferralsToBeCancelled.length !== 0}
                onSendRequest={requestCancelDeferral}
                onCompleteRequest={completeCancelDeferral}
                onCancel={cancelCancellation}
            />
        </>
    );
}

export default DeferredCVEsTable;
