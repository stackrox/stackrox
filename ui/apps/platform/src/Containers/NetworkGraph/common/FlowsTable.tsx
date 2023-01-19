import React, { ReactElement } from 'react';
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { Flex, FlexItem, Text, TextContent, TextVariants } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import { Flow } from '../types/flow.type';
import { protocolLabel } from '../utils/flowUtils';

type FlowsTableProps = {
    label: string;
    flows: Flow[];
    numFlows: number;
    expandedRows: string[];
    setExpandedRows: React.Dispatch<React.SetStateAction<string[]>>;
    selectedRows: string[];
    setSelectedRows: React.Dispatch<React.SetStateAction<string[]>>;
    isEditable?: boolean;
    addToBaseline?: (flow: Flow) => void;
    markAsAnomalous?: (flow: Flow) => void;
};

const columnNames = {
    entity: 'Entity',
    direction: 'Direction',
    // @TODO: This would be a good point to update with i18n translation ability
    portAndProtocol: 'Port / protocol',
};

function FlowsTable({
    label,
    flows,
    numFlows,
    expandedRows,
    setExpandedRows,
    selectedRows,
    setSelectedRows,
    isEditable = false,
    addToBaseline,
    markAsAnomalous,
}: FlowsTableProps): ReactElement {
    // getter functions
    const isRowExpanded = (row: Flow) => expandedRows.includes(row.id);
    const areAllRowsSelected = selectedRows.length !== 0 && selectedRows.length === numFlows;
    const isRowSelected = (row: Flow) => selectedRows.includes(row.id);

    // setter functions
    const setRowExpanded = (row: Flow, isExpanding = true) =>
        setExpandedRows((prevExpanded) => {
            const otherExpandedRows = prevExpanded.filter((r) => r !== row.id);
            return isExpanding ? [...otherExpandedRows, row.id] : otherExpandedRows;
        });
    const setRowSelected = (row: Flow, isSelecting = true) =>
        setSelectedRows((prevSelected) => {
            const otherSelectedRows = prevSelected.filter((r) => r !== row.id);
            return isSelecting ? [...otherSelectedRows, row.id] : otherSelectedRows;
        });
    const selectAllRows = (isSelecting = true) => {
        if (isSelecting) {
            const newSelectedRows = flows.reduce((acc, curr) => {
                if (curr.children && curr.children.length !== 0) {
                    return [...acc, ...curr.children.map((child) => child.id)];
                }
                return [...acc, curr.id];
            }, [] as string[]);
            return setSelectedRows(newSelectedRows);
        }
        return setSelectedRows([]);
    };

    return (
        <TableComposable aria-label={label} variant="compact">
            <Thead>
                <Tr>
                    <Th />
                    {isEditable && (
                        <Th
                            select={{
                                onSelect: (_event, isSelecting) => selectAllRows(isSelecting),
                                isSelected: areAllRowsSelected,
                            }}
                        />
                    )}
                    <Th width={40}>{columnNames.entity}</Th>
                    <Th>{columnNames.direction}</Th>
                    <Th>{columnNames.portAndProtocol}</Th>
                    <Th />
                </Tr>
            </Thead>
            {flows.map((row, rowIndex) => {
                const isExpanded = isRowExpanded(row);
                const rowActions: IAction[] =
                    row.children && !row.children.length
                        ? [
                              row.isAnomalous
                                  ? {
                                        itemKey: 'add-flow-to-baseline',
                                        title: 'Add to baseline',
                                        onClick: () => {
                                            addToBaseline?.(row);
                                        },
                                    }
                                  : {
                                        itemKey: 'mark-flow-as-anomalous',
                                        title: 'Mark as anomalous',
                                        onClick: () => {
                                            markAsAnomalous?.(row);
                                        },
                                    },
                          ]
                        : [];

                return (
                    <Tbody key={row.id} isExpanded={isExpanded}>
                        <Tr>
                            <Td
                                expand={
                                    row.children && row.children.length
                                        ? {
                                              rowIndex,
                                              isExpanded,
                                              onToggle: () => setRowExpanded(row, !isExpanded),
                                              expandId: 'flow-expandable',
                                          }
                                        : undefined
                                }
                            />
                            {isEditable && (
                                <Td
                                    select={
                                        row.children && row.children.length === 0
                                            ? {
                                                  rowIndex,
                                                  onSelect: (_event, isSelecting) =>
                                                      setRowSelected(row, isSelecting),
                                                  isSelected: isRowSelected(row),
                                                  disable: !!row.children.length,
                                              }
                                            : undefined
                                    }
                                />
                            )}
                            <Td dataLabel={columnNames.entity}>
                                <Flex direction={{ default: 'row' }}>
                                    <FlexItem>
                                        <div>{row.entity}</div>
                                        <div>
                                            <TextContent>
                                                <Text component={TextVariants.small}>
                                                    {row.type === 'DEPLOYMENT'
                                                        ? `in "${row.namespace}"`
                                                        : 'External to cluster'}
                                                </Text>
                                            </TextContent>
                                        </div>
                                    </FlexItem>
                                    {row.isAnomalous && (
                                        <FlexItem>
                                            <ExclamationCircleIcon className="pf-u-danger-color-100" />
                                        </FlexItem>
                                    )}
                                </Flex>
                            </Td>
                            <Td dataLabel={columnNames.direction}>{row.direction}</Td>
                            <Td dataLabel={columnNames.portAndProtocol}>
                                {row.port} / {protocolLabel[row.protocol]}
                            </Td>
                            {isEditable && (
                                <Td isActionCell>
                                    {row.children && !row.children.length && (
                                        <ActionsColumn items={rowActions} />
                                    )}
                                </Td>
                            )}
                        </Tr>
                        {isExpanded &&
                            row.children &&
                            row.children.map((child) => {
                                const childActions: IAction[] = [
                                    child.isAnomalous
                                        ? {
                                              itemKey: 'add-flow-to-baseline',
                                              title: 'Add to baseline',
                                              onClick: () => {
                                                  addToBaseline?.(child);
                                              },
                                          }
                                        : {
                                              itemKey: 'mark-flow-as-anomalous',
                                              title: 'Mark as anomalous',
                                              onClick: () => {
                                                  markAsAnomalous?.(child);
                                              },
                                          },
                                ];

                                return (
                                    <Tr key={child.id} isExpanded={isExpanded}>
                                        <Td />
                                        {isEditable && (
                                            <Td
                                                select={{
                                                    rowIndex,
                                                    onSelect: (_event, isSelecting) =>
                                                        setRowSelected(child, isSelecting),
                                                    isSelected: isRowSelected(child),
                                                }}
                                            />
                                        )}
                                        <Td>
                                            <ExpandableRowContent>
                                                <Flex direction={{ default: 'row' }}>
                                                    <FlexItem>{child.entity}</FlexItem>
                                                    {row.isAnomalous && (
                                                        <FlexItem>
                                                            <ExclamationCircleIcon className="pf-u-danger-color-100" />
                                                        </FlexItem>
                                                    )}
                                                </Flex>
                                            </ExpandableRowContent>
                                        </Td>
                                        <Td>
                                            <ExpandableRowContent>
                                                {child.direction}
                                            </ExpandableRowContent>
                                        </Td>
                                        <Td>
                                            <ExpandableRowContent>
                                                {child.port} / {protocolLabel[child.protocol]}
                                            </ExpandableRowContent>
                                        </Td>
                                        {isEditable && (
                                            <Td isActionCell>
                                                <ActionsColumn items={childActions} />
                                            </Td>
                                        )}
                                    </Tr>
                                );
                            })}
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default FlowsTable;
