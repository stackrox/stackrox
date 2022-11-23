import React from 'react';
import {
    Divider,
    DropdownItem,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Text,
    TextContent,
    TextVariants,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
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
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';

import './DeploymentFlows.css';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import { Flow, FlowBase } from '../types';
import { getAllUniquePorts } from '../utils/flowUtils';

const columnNames = {
    entity: 'Entity',
    direction: 'Direction',
    portAndProtocol: 'Port / protocol',
};

const flows: Flow[] = [
    {
        id: 'External Entities-Ingress-Many-TCP',
        type: 'External',
        entity: 'External Entities',
        namespace: '',
        direction: 'Ingress',
        port: 'Many',
        protocol: 'TCP',
        isAnomalous: true,
        children: [
            {
                id: 'External Entities-Ingress-443-TCP',
                type: 'External',
                entity: 'External Entities',
                namespace: '',
                direction: 'Ingress',
                port: '443',
                protocol: 'TCP',
                isAnomalous: true,
            },
            {
                id: 'External Entities-Ingress-9443-TCP',
                type: 'External',
                entity: 'External Entities',
                namespace: '',
                direction: 'Ingress',
                port: '9443',
                protocol: 'TCP',
                isAnomalous: true,
            },
        ],
    },
    {
        id: 'Deployment 1-naples-Ingress-Many-TCP',
        type: 'Deployment',
        entity: 'Deployment 1',
        namespace: 'naples',
        direction: 'Ingress',
        port: '9000',
        protocol: 'TCP',
        isAnomalous: true,
        children: [],
    },
    {
        id: 'Deployment 2-naples-Ingress-Many-UDP',
        type: 'Deployment',
        entity: 'Deployment 2',
        namespace: 'naples',
        direction: 'Ingress',
        port: '8080',
        protocol: 'UDP',
        isAnomalous: false,
        children: [],
    },
    {
        id: 'Deployment 3-naples-Egress-7777-UDP',
        type: 'Deployment',
        entity: 'Deployment 3',
        namespace: 'naples',
        direction: 'Egress',
        port: '7777',
        protocol: 'UDP',
        isAnomalous: false,
        children: [],
    },
];

function DeploymentFlow() {
    // component state
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );
    const initialExpandedRows = flows.filter((row) => !!row.children.length).map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = React.useState<string[]>([]);

    // derived data
    const totalFlows = flows.reduce((acc, curr) => {
        // if there are no children then it counts as 1 flow
        return acc + (curr.children.length ? curr.children.length : 1);
    }, 0);
    const allUniquePorts = getAllUniquePorts(flows);

    // getter functions
    const isRowExpanded = (row: Flow) => expandedRows.includes(row.id);
    const areAllRowsSelected = selectedRows.length === totalFlows;
    const isRowSelected = (row: Flow | FlowBase) => selectedRows.includes(row.id);

    // setter functions
    const setRowExpanded = (row: Flow, isExpanding = true) =>
        setExpandedRows((prevExpanded) => {
            const otherExpandedRows = prevExpanded.filter((r) => r !== row.id);
            return isExpanding ? [...otherExpandedRows, row.id] : otherExpandedRows;
        });
    const setRowSelected = (row: Flow | FlowBase, isSelecting = true) =>
        setSelectedRows((prevSelected) => {
            const otherSelectedRows = prevSelected.filter((r) => r !== row.id);
            return isSelecting ? [...otherSelectedRows, row.id] : otherSelectedRows;
        });
    const selectAllRows = (isSelecting = true) => {
        if (isSelecting) {
            const newSelectedRows = flows.reduce((acc, curr) => {
                if (curr.children.length !== 0) {
                    return [...acc, ...curr.children.map((child) => child.id)];
                }
                return [...acc, curr.id];
            }, [] as string[]);
            return setSelectedRows(newSelectedRows);
        }
        return setSelectedRows([]);
    };
    const markSelectedAsAnomalous = () => {
        // @TODO: Mark as anomalous
        setSelectedRows([]);
    };
    const addSelectedToBaseline = () => {
        // @TODO: Add to baseline
        setSelectedRows([]);
    };

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <Flex>
                        <FlexItem flex={{ default: 'flex_1' }}>
                            <EntityNameSearchInput
                                value={entityNameFilter}
                                setValue={setEntityNameFilter}
                            />
                        </FlexItem>
                        <FlexItem>
                            <AdvancedFlowsFilter
                                filters={advancedFilters}
                                setFilters={setAdvancedFilters}
                                allUniquePorts={allUniquePorts}
                            />
                        </FlexItem>
                    </Flex>
                </StackItem>
                <Divider component="hr" />
                <StackItem>
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem>
                                <TextContent>
                                    <Text component={TextVariants.h3}>
                                        {totalFlows} active flows
                                    </Text>
                                </TextContent>
                            </ToolbarItem>
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <BulkActionsDropdown isDisabled={selectedRows.length === 0}>
                                    <DropdownItem
                                        key="mark_as_anomalous"
                                        component="button"
                                        onClick={markSelectedAsAnomalous}
                                    >
                                        Mark as anomalous
                                    </DropdownItem>
                                    <DropdownItem
                                        key="add_to_baseline"
                                        component="button"
                                        onClick={addSelectedToBaseline}
                                    >
                                        Add to baseline
                                    </DropdownItem>
                                </BulkActionsDropdown>
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <StackItem>
                    <TableComposable aria-label="Deployment flow" variant="compact">
                        <Thead>
                            <Tr>
                                <Th />
                                <Th
                                    select={{
                                        onSelect: (_event, isSelecting) =>
                                            selectAllRows(isSelecting),
                                        isSelected: areAllRowsSelected,
                                    }}
                                />
                                <Th width={40}>{columnNames.entity}</Th>
                                <Th>{columnNames.direction}</Th>
                                <Th>{columnNames.portAndProtocol}</Th>
                                <Th />
                            </Tr>
                        </Thead>
                        {flows.map((row, rowIndex) => {
                            const isExpanded = isRowExpanded(row);
                            const rowActions: IAction[] = !row.children.length
                                ? [
                                      row.isAnomalous
                                          ? {
                                                itemKey: 'add-flow-to-baseline',
                                                title: 'Add to baseline',
                                                onClick: () => {},
                                            }
                                          : {
                                                itemKey: 'mark-flow-as-anomalous',
                                                title: 'Mark as anomalous',
                                                onClick: () => {},
                                            },
                                  ]
                                : [];

                            return (
                                <Tbody key={row.id} isExpanded={isExpanded}>
                                    <Tr>
                                        <Td
                                            expand={
                                                row.children.length
                                                    ? {
                                                          rowIndex,
                                                          isExpanded,
                                                          onToggle: () =>
                                                              setRowExpanded(row, !isExpanded),
                                                          expandId: 'flow-expandable',
                                                      }
                                                    : undefined
                                            }
                                        />
                                        <Td
                                            select={
                                                row.children.length === 0
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
                                        <Td dataLabel={columnNames.entity}>
                                            <Flex direction={{ default: 'row' }}>
                                                <FlexItem>
                                                    <div>{row.entity}</div>
                                                    <div>
                                                        <TextContent>
                                                            <Text component={TextVariants.small}>
                                                                {row.type === 'Deployment'
                                                                    ? `in "${row.namespace}"`
                                                                    : `${row.children.length} active flows`}
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
                                            {row.port} / {row.protocol}
                                        </Td>
                                        <Td isActionCell>
                                            {!row.children.length && (
                                                <ActionsColumn items={rowActions} />
                                            )}
                                        </Td>
                                    </Tr>
                                    {isExpanded &&
                                        row.children.map((child) => {
                                            const childActions: IAction[] = [
                                                child.isAnomalous
                                                    ? {
                                                          itemKey: 'add-flow-to-baseline',
                                                          title: 'Add to baseline',
                                                          onClick: () => {},
                                                      }
                                                    : {
                                                          itemKey: 'mark-flow-as-anomalous',
                                                          title: 'Mark as anomalous',
                                                          onClick: () => {},
                                                      },
                                            ];

                                            return (
                                                <Tr key={child.id} isExpanded={isExpanded}>
                                                    <Td />
                                                    <Td
                                                        select={{
                                                            rowIndex,
                                                            onSelect: (_event, isSelecting) =>
                                                                setRowSelected(child, isSelecting),
                                                            isSelected: isRowSelected(child),
                                                        }}
                                                    />
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
                                                            {child.port} / {child.protocol}
                                                        </ExpandableRowContent>
                                                    </Td>
                                                    <Td isActionCell>
                                                        <ActionsColumn items={childActions} />
                                                    </Td>
                                                </Tr>
                                            );
                                        })}
                                </Tbody>
                            );
                        })}
                    </TableComposable>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlow;
