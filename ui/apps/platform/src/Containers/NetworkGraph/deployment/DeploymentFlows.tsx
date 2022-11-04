import React from 'react';
import {
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
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import './DeploymentFlows.css';

interface FlowBase {
    type: 'Deployment' | 'External';
    entity: string;
    namespace: string;
    direction: string;
    port: string;
    protocol: string;
    isAnomalous: boolean;
}

interface Flow extends FlowBase {
    children: FlowBase[];
}

const columnNames = {
    entity: 'Entity',
    direction: 'Direction',
    portAndProtocol: 'Port / protocol',
};

const flows: Flow[] = [
    {
        type: 'External',
        entity: 'External Entities',
        namespace: '',
        direction: 'Ingress',
        port: 'Many',
        protocol: 'TCP',
        isAnomalous: true,
        children: [
            {
                type: 'External',
                entity: 'External Entities',
                namespace: '',
                direction: 'Ingress',
                port: '443',
                protocol: 'TCP',
                isAnomalous: true,
            },
            {
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
        type: 'Deployment',
        entity: 'Deployment 1',
        namespace: 'naples',
        direction: 'Ingress',
        port: 'Many',
        protocol: 'TCP',
        isAnomalous: true,
        children: [],
    },
    {
        type: 'Deployment',
        entity: 'Deployment 2',
        namespace: 'naples',
        direction: 'Ingress',
        port: 'Many',
        protocol: 'UDP',
        isAnomalous: false,
        children: [],
    },
    {
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
    const initialExpandedRows = flows
        .filter((row) => !!row.children.length)
        .map((row) => row.entity); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);

    const setRowExpanded = (row: Flow, isExpanding = true) =>
        setExpandedRows((prevExpanded) => {
            const otherExpandedRows = prevExpanded.filter((r) => r !== row.entity);
            return isExpanding ? [...otherExpandedRows, row.entity] : otherExpandedRows;
        });

    const isRowExpanded = (row: Flow) => expandedRows.includes(row.entity);

    const totalFlows = flows.reduce((acc, curr) => {
        // if there are no children then it counts as 1 flow
        return acc + (curr.children.length ? curr.children.length : 1);
    }, 0);

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
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
                            <ToolbarItem alignment={{ default: 'alignRight' }} />
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <StackItem>
                    <TableComposable aria-label="Deployment flow" variant="compact">
                        <Thead>
                            <Tr>
                                <Th width={10} />
                                <Th width={40}>{columnNames.entity}</Th>
                                <Th width={20}>{columnNames.direction}</Th>
                                <Th width={30}>{columnNames.portAndProtocol}</Th>
                            </Tr>
                        </Thead>
                        {flows.map((row, rowIndex) => {
                            const isExpanded = isRowExpanded(row);
                            return (
                                <Tbody key={row.entity} isExpanded={isExpanded}>
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
                                    </Tr>
                                    {isExpanded &&
                                        row.children.map((child) => {
                                            return (
                                                <Tr key={child.entity} isExpanded={isExpanded}>
                                                    <Td />
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
