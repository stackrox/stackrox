import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
    Button,
    Divider,
    Dropdown,
    DropdownItem,
    DropdownSeparator,
    DropdownToggle,
    Flex,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { CaretDownIcon, CheckCircleIcon } from '@patternfly/react-icons';
import orderBy from 'lodash/orderBy';

import { ListPolicy } from 'types/policy.proto';
import { sortSeverity, sortAsciiCaseInsensitive, sortValueByLength } from 'sorters/sorters';
import TableCell from 'Components/PatternFly/TableCell';
import { ActionItem } from 'Containers/Violations/PatternFly/ViolationsTablePanel';
import useTableSelection from 'hooks/useTableSelection';
import { SortDirection } from 'hooks/useTableSort';
import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';

import { formatLifecycleStages } from './policies.utils';
import PolicySeverityLabel from './PolicySeverityLabel';

const columns = [
    {
        Header: 'Policy',
        accessor: 'name',
        Cell: ({ original, value }) => (
            <Button
                variant="link"
                isInline
                component={(props) => (
                    <Link {...props} to={`${policiesBasePath}/${original.id as string}`} />
                )}
            >
                {value}
            </Button>
        ),
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortAsciiCaseInsensitive(a.name, b.name),
        width: 20 as const,
    },
    {
        Header: 'Description',
        accessor: 'description',
        width: 40 as const,
    },
    {
        Header: 'Status',
        accessor: 'disabled',
        Cell: ({ value }) => {
            if (value) {
                return 'Disabled';
            }
            return (
                <Flex className="pf-u-info-color-200">
                    <CheckCircleIcon className="pf-u-mr-sm pf-m-align-self-center" />
                    Enabled
                </Flex>
            );
        },
        width: 15 as const,
    },
    {
        Header: 'Notifiers',
        accessor: 'notifiers',
        Cell: ({ value }) => {
            return value.join(', ') as string;
        },
        sortMethod: (a: ListPolicy, b: ListPolicy) => sortValueByLength(a.notifiers, b.notifiers),
    },
    {
        Header: 'Severity',
        accessor: 'severity',
        Cell: ({ value }) => {
            return <PolicySeverityLabel severity={value} />;
        },
        sortMethod: (a: ListPolicy, b: ListPolicy) => -sortSeverity(a.severity, b.severity),
    },
    {
        Header: 'Lifecycle',
        accessor: 'lifecycleStages',
        Cell: ({ value }) => {
            return formatLifecycleStages(value);
        },
    },
];

type PoliciesTableProps = {
    policies?: ListPolicy[];
    hasWriteAccessForPolicy: boolean;
    deletePoliciesHandler: (ids) => void;
    exportPoliciesHandler: (ids, onClearAll?) => void;
    enablePoliciesHandler: (ids) => void;
    disablePoliciesHandler: (ids) => void;
    pageActionButtons: React.ReactElement;
};

function PoliciesTable({
    policies = [],
    hasWriteAccessForPolicy,
    deletePoliciesHandler,
    exportPoliciesHandler,
    enablePoliciesHandler,
    disablePoliciesHandler,
    pageActionButtons,
}: PoliciesTableProps): React.ReactElement {
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(0);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] = useState<SortDirection>('asc');
    // Handle Bulk Actions dropdown state.
    const [isActionsOpen, setIsActionsOpen] = useState(false);
    // For sorting data client side
    const [rows, setRows] = useState<ListPolicy[]>([]);

    // a map to keep track of row index within the table to the policy id
    // for checkbox selection after the table has been sorted
    const rowIdToIndex = {};
    policies.forEach(({ id }, idx) => {
        rowIdToIndex[id] = idx;
    });

    // Handle selected rows in table
    const {
        selected,
        allRowsSelected,
        hasSelections,
        onSelect,
        onSelectAll,
        // onClearAll,
        getSelectedIds,
    } = useTableSelection(policies);

    function onToggleActions(toggleOpen) {
        setIsActionsOpen(toggleOpen);
    }

    function onSelectActions() {
        setIsActionsOpen(false);
    }

    function onSort(e, index, direction) {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
    }

    const selectedIds = getSelectedIds();
    const selectedPolicies = policies.filter(({ id }) => selectedIds.includes(id));
    let numEnabled = 0;
    let numDisabled = 0;
    let numDeletable = 0;
    selectedPolicies.forEach(({ disabled, isDefault }) => {
        if (disabled) {
            numDisabled += 1;
        } else {
            numEnabled += 1;
        }
        if (!isDefault) {
            numDeletable += 1;
        }
    });

    // If we use server side page management, this becomes unnecessary
    useEffect(() => {
        const { sortMethod, accessor } = columns[activeSortIndex];
        let sortedPolicies = [...policies];
        if (sortMethod) {
            sortedPolicies.sort(sortMethod);
            if (activeSortDirection === 'desc') {
                sortedPolicies.reverse();
            }
        } else {
            sortedPolicies = orderBy(sortedPolicies, [accessor], [activeSortDirection]);
        }
        setRows(sortedPolicies);
    }, [policies, activeSortIndex, activeSortDirection]);

    // TODO: https://stack-rox.atlassian.net/browse/ROX-8613
    // isDisabled={!hasSelections}
    // dropdownItems={hasWriteAccessForPolicy ? [Enable, Disable, Export, Delete] : [Export]} see PolicyDetail.tsx
    return (
        <>
            <Toolbar inset={{ default: 'insetNone' }}>
                <ToolbarContent>
                    <ToolbarItem>
                        <Title headingLevel="h2" className="pf-u-color-100 pf-u-ml-sm">
                            Policies
                        </Title>
                    </ToolbarItem>
                    <ToolbarItem>{pageActionButtons}</ToolbarItem>
                    <ToolbarItem>
                        <Dropdown
                            data-testid="policies-bulk-actions-dropdown"
                            onSelect={onSelectActions}
                            toggle={
                                <DropdownToggle
                                    isDisabled={!hasWriteAccessForPolicy || !hasSelections}
                                    isPrimary
                                    onToggle={onToggleActions}
                                    toggleIndicator={CaretDownIcon}
                                >
                                    Actions
                                </DropdownToggle>
                            }
                            isOpen={isActionsOpen}
                            dropdownItems={[
                                <DropdownItem
                                    key="Enable policies"
                                    component="button"
                                    isDisabled={numDisabled === 0}
                                    onClick={() => enablePoliciesHandler(selectedIds)}
                                >
                                    {`Enable policies (${numDisabled})`}
                                </DropdownItem>,
                                <DropdownItem
                                    key="Disable policies"
                                    component="button"
                                    isDisabled={numEnabled === 0}
                                    onClick={() => disablePoliciesHandler(selectedIds)}
                                >
                                    {`Disable policies (${numEnabled})`}
                                </DropdownItem>,
                                // TODO: https://stack-rox.atlassian.net/browse/ROX-8613
                                // Export policies to JSON
                                // onClick={() => exportPoliciesHandler(selectedIds, onClearAll)}
                                // {`Export policies to JSON (${numSelected})`}
                                <DropdownSeparator key="Separator" />,
                                <DropdownItem
                                    key="Delete policy"
                                    component="button"
                                    isDisabled={numDeletable === 0}
                                    onClick={() =>
                                        deletePoliciesHandler(
                                            selectedPolicies
                                                .filter(({ isDefault }) => !isDefault)
                                                .map(({ id }) => id)
                                        )
                                    }
                                >
                                    {`Delete policies (${numDeletable})`}
                                </DropdownItem>,
                            ]}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>
            <Divider component="div" />
            <PageSection isFilled padding={{ default: 'noPadding' }} hasOverflowScroll>
                <TableComposable>
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            {columns.map(({ Header, width }, columnIndex) => {
                                const sortParams = {
                                    sort: {
                                        sortBy: {
                                            index: activeSortIndex,
                                            direction: activeSortDirection,
                                        },
                                        onSort,
                                        columnIndex,
                                    },
                                };
                                return (
                                    <Th key={Header} modifier="wrap" width={width} {...sortParams}>
                                        {Header}
                                    </Th>
                                );
                            })}
                            <Th />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {rows.map((policy) => {
                            const { disabled, id, isDefault } = policy;
                            let togglePolicyAction = {} as ActionItem;
                            if (disabled) {
                                togglePolicyAction = {
                                    title: 'Enable policy',
                                    onClick: () => enablePoliciesHandler([id]),
                                };
                            } else {
                                togglePolicyAction = {
                                    title: 'Disable policy',
                                    onClick: () => disablePoliciesHandler([id]),
                                };
                            }
                            const exportPolicyAction = {
                                title: 'Export policy to JSON',
                                onClick: () => exportPoliciesHandler([id]),
                            };
                            const deletePolicyAction = {
                                title: 'Delete policy',
                                onClick: () => deletePoliciesHandler([id]),
                                disabled: isDefault,
                            };
                            const actionItems: ActionItem[] = [
                                togglePolicyAction,
                                exportPolicyAction,
                                deletePolicyAction,
                            ];
                            const rowIndex = rowIdToIndex[id];
                            return (
                                <Tr key={id}>
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    {columns.map((column) => {
                                        return (
                                            <TableCell
                                                key={column.Header}
                                                row={policy}
                                                column={column}
                                            />
                                        );
                                    })}
                                    <Td
                                        actions={{
                                            items: actionItems,
                                        }}
                                    />
                                </Tr>
                            );
                        })}
                    </Tbody>
                </TableComposable>
            </PageSection>
        </>
    );
}

export default PoliciesTable;
