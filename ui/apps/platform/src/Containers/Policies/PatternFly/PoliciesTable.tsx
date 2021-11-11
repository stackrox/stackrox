import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import {
    Flex,
    FlexItem,
    Divider,
    PageSection,
    Title,
    Badge,
    Button,
    Select,
    SelectOption,
    Label,
} from '@patternfly/react-core';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { CheckCircleIcon } from '@patternfly/react-icons';
import orderBy from 'lodash/orderBy';

import { ListPolicy } from 'types/policy.proto';
import { sortSeverity, sortAsciiCaseInsensitive, sortValueByLength } from 'sorters/sorters';
import { severityColorMapPF } from 'constants/severityColors';
import { severityLabels, lifecycleStageLabels } from 'messages/common';
import TableCell from 'Components/PatternFly/TableCell';
import { ActionItem } from 'Containers/Violations/PatternFly/ViolationsTablePanel';
import useTableSelection from 'hooks/useTableSelection';
import { SortDirection } from 'hooks/useTableSort';
import { policiesBasePathPatternFly as policiesBasePath } from 'routePaths';

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
            const severity = severityLabels[value];
            return <Label color={severityColorMapPF[severity]}>{severity}</Label>;
        },
        sortMethod: (a: ListPolicy, b: ListPolicy) => -sortSeverity(a.severity, b.severity),
    },
    {
        Header: 'Lifecycle',
        accessor: 'lifecycleStages',
        Cell: ({ value }) => {
            return value.map((stage) => lifecycleStageLabels[stage] as string).join(', ') as string;
        },
    },
];

type PoliciesTableProps = {
    policies?: ListPolicy[];
    deletePoliciesHandler: (id) => void;
    exportPoliciesHandler: (id) => void;
    enablePoliciesHandler: (ids) => void;
    disablePoliciesHandler: (id) => void;
};

function PoliciesTable({
    policies = [],
    deletePoliciesHandler,
    exportPoliciesHandler,
    enablePoliciesHandler,
    disablePoliciesHandler,
}: PoliciesTableProps): React.ReactElement {
    // index of the currently active column
    const [activeSortIndex, setActiveSortIndex] = useState(0);
    // sort direction of the currently active column
    const [activeSortDirection, setActiveSortDirection] = useState<SortDirection>('asc');
    // Handle Bulk Actions dropdown state.
    const [isSelectOpen, setIsSelectOpen] = useState(false);
    // For sorting data client side
    const [rows, setRows] = useState<ListPolicy[]>([]);

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

    function onToggleSelect(toggleOpen) {
        setIsSelectOpen(toggleOpen);
    }

    function closeSelect() {
        setIsSelectOpen(false);
    }

    function onSort(e, index, direction) {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
    }

    const selectedIds = getSelectedIds();
    const numSelected = selectedIds.length;
    const selectedPolicies = policies.filter(({ id }) => selectedIds.includes(id));
    let numEnabled = 0;
    let numDisabled = 0;
    selectedPolicies.forEach(({ disabled }) => {
        if (disabled) {
            numDisabled += 1;
        } else {
            numEnabled += 1;
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

    return (
        <>
            <Flex
                className="pf-u-p-md"
                alignSelf={{ default: 'alignSelfCenter' }}
                fullWidth={{ default: 'fullWidth' }}
            >
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                    <Title headingLevel="h2" className="pf-u-color-100 pf-u-ml-sm">
                        Policies
                    </Title>
                </FlexItem>
                <FlexItem alignSelf={{ default: 'alignSelfCenter' }}>
                    <Badge isRead>{policies.length}</Badge>
                </FlexItem>
                <FlexItem data-testid="policies-bulk-actions-dropdown">
                    {/* TODO: will address this in ROX-8355 */}
                    <Select
                        onToggle={onToggleSelect}
                        isOpen={isSelectOpen}
                        placeholderText="Bulk Actions"
                        onSelect={closeSelect}
                        isDisabled={!hasSelections}
                    >
                        <SelectOption
                            key="0"
                            value={`Enable policies (${numDisabled})`}
                            onClick={() => enablePoliciesHandler(selectedIds)}
                            data-testid="bulk-add-tags-btn"
                        />
                        <SelectOption
                            key="0"
                            value={`Disable policies (${numEnabled})`}
                            onClick={() => disablePoliciesHandler(selectedIds)}
                            data-testid="bulk-add-tags-btn"
                        />
                        <SelectOption
                            key="1"
                            value={`Export to JSON (${numSelected})`}
                            onClick={() => exportPoliciesHandler(selectedIds)}
                        />
                        <SelectOption
                            key="2"
                            value={`Delete (${numSelected})`}
                            onClick={() => deletePoliciesHandler(selectedIds)}
                        />
                    </Select>
                </FlexItem>
            </Flex>
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
                        {rows.map((policy, rowIndex) => {
                            const { disabled, id } = policy;
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
                                title: 'Export to JSON',
                                onClick: () => exportPoliciesHandler([id]),
                            };
                            const deletePolicyAction = {
                                title: 'Delete',
                                onClick: () => deletePoliciesHandler([id]),
                            };
                            const actionItems: ActionItem[] = [
                                togglePolicyAction,
                                exportPolicyAction,
                                deletePolicyAction,
                            ];
                            return (
                                <Tr key={policy.id}>
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
