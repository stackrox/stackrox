import React, { ReactElement } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import resolvePath from 'object-resolve-path';

import { resolveAlert } from 'services/AlertsService';
import { excludeDeployments } from 'services/PoliciesService';
import { ENFORCEMENT_ACTIONS } from 'constants/enforcementActions';
import VIOLATION_STATES from 'constants/violationStates';
import LIFECYCLE_STAGES from 'constants/lifecycleStages';
import { TableColumn, SortDirection } from 'hooks/useTableSort';
import { Violation } from './types/violationTypes';

type TableCellProps = {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    row: Violation;
    column: TableColumn;
};

function TableCell({ row, column }: TableCellProps): React.ReactElement {
    let value = resolvePath(row, column.accessor);
    if (column.Cell) {
        value = column.Cell({ original: row, value });
    }
    return <Td key={column.Header}>{value || '-'}</Td>;
}

type ActionItem = {
    title: string | ReactElement;
    onClick: (item) => void;
};

type ViolationsTableProps = {
    violations: Violation[];
    columns: TableColumn[];
    setSelectedAlertId: (id) => void;
    selected: {
        [key: number]: boolean;
    };
    allRowsSelected: boolean;
    onSelect: (e, isSelected, rowId) => void;
    onSelectAll: (e, isSelected) => void;
    activeSortIndex: number;
    setActiveSortIndex: (idx) => void;
    activeSortDirection: SortDirection;
    setActiveSortDirection: (dir) => void;
};

function ViolationsTable({
    violations,
    columns,
    setSelectedAlertId,
    selected,
    allRowsSelected,
    onSelect,
    onSelectAll,
    activeSortIndex,
    setActiveSortIndex,
    activeSortDirection,
    setActiveSortDirection,
}: ViolationsTableProps): ReactElement {
    function resolveAlertAction(addToBaseline, violation) {
        const unselectAlert = () => setSelectedAlertId(null);
        return () => {
            resolveAlert(violation.id, addToBaseline).then(unselectAlert, unselectAlert);
        };
    }
    function onSort(e, index, direction) {
        setActiveSortIndex(index);
        setActiveSortDirection(direction);
    }

    function onRowClick(violation) {
        return () => {
            setSelectedAlertId(violation.id);
        };
    }

    return (
        <TableComposable variant="compact" isStickyHeader>
            <Thead>
                <Tr>
                    <Th
                        select={{
                            onSelect: onSelectAll,
                            isSelected: allRowsSelected,
                        }}
                    />
                    {columns.map(({ Header, sortField }, idx) => {
                        const sortParams = sortField
                            ? {
                                  sort: {
                                      sortBy: {
                                          index: activeSortIndex,
                                          direction: activeSortDirection,
                                      },
                                      onSort,
                                      columnIndex: idx,
                                  },
                              }
                            : {};
                        return (
                            <Th modifier="wrap" {...sortParams}>
                                {Header}
                            </Th>
                        );
                    })}
                    <Th />
                </Tr>
            </Thead>
            <Tbody>
                {violations.map((violation, rowIndex) => {
                    const {
                        state,
                        lifecycleStage,
                        enforcementAction,
                        deployment,
                        policy,
                        id,
                    } = violation;
                    const isAttemptedViolation = state === VIOLATION_STATES.ATTEMPTED;
                    const isResolved = state === VIOLATION_STATES.RESOLVED;
                    const isRuntimeAlert = lifecycleStage === LIFECYCLE_STAGES.RUNTIME;
                    const isDeployCreateAttemptedAlert =
                        enforcementAction ===
                        ENFORCEMENT_ACTIONS.FAIL_DEPLOYMENT_CREATE_ENFORCEMENT;

                    const actionItems: ActionItem[] = [];
                    if (!isResolved) {
                        if (isRuntimeAlert) {
                            actionItems.push({
                                title: 'Resolve and add to process baseline',
                                onClick: () => resolveAlertAction(true, violation),
                            });
                        }
                        if (isRuntimeAlert || isAttemptedViolation) {
                            actionItems.push({
                                title: 'Mark as resolved',
                                onClick: () => resolveAlertAction(false, violation),
                            });
                        }
                    }
                    if (!isDeployCreateAttemptedAlert && deployment?.name) {
                        actionItems.push({
                            title: 'Exclude deployment',
                            onClick: () => excludeDeployments(policy.id, [deployment.name]),
                        });
                    }
                    return (
                        <Tr key={id} onClick={onRowClick(violation)}>
                            <Td
                                key={id}
                                select={{
                                    rowIndex,
                                    onSelect,
                                    isSelected: selected[rowIndex],
                                }}
                            />
                            {columns.map((column) => {
                                return <TableCell row={violation} column={column} />;
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
    );
}

export default ViolationsTable;
