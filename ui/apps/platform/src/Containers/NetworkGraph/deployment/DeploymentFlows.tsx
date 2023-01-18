import React from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { Flow } from '../types/flow.type';
import { getAllUniquePorts, getNumFlows } from '../utils/flowUtils';

import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import FlowsTable from '../common/FlowsTable';
import FlowsTableHeaderText from '../common/FlowsTableHeaderText';
import FlowsBulkActions from '../common/FlowsBulkActions';

import './DeploymentFlows.css';

const flows: Flow[] = [
    {
        id: 'External Entities-Ingress-Many-TCP',
        type: 'EXTERNAL_ENTITIES',
        entity: 'External Entities',
        entityId: '12345',
        namespace: '',
        direction: 'Ingress',
        port: 'Many',
        protocol: 'TCP',
        isAnomalous: true,
        children: [
            {
                id: 'External Entities-Ingress-443-TCP',
                type: 'EXTERNAL_ENTITIES',
                entity: 'External Entities',
                entityId: '12345',
                namespace: '',
                direction: 'Ingress',
                port: '443',
                protocol: 'TCP',
                isAnomalous: true,
            },
            {
                id: 'External Entities-Ingress-9443-TCP',
                type: 'EXTERNAL_ENTITIES',
                entityId: '12345',
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
        type: 'DEPLOYMENT',
        entity: 'Deployment 1',
        entityId: '00000',
        namespace: 'naples',
        direction: 'Ingress',
        port: '9000',
        protocol: 'TCP',
        isAnomalous: true,
        children: [],
    },
    {
        id: 'Deployment 2-naples-Ingress-Many-UDP',
        type: 'DEPLOYMENT',
        entity: 'Deployment 2',
        entityId: '11111',
        namespace: 'naples',
        direction: 'Ingress',
        port: '8080',
        protocol: 'UDP',
        isAnomalous: false,
        children: [],
    },
    {
        id: 'Deployment 3-naples-Egress-7777-UDP',
        type: 'DEPLOYMENT',
        entity: 'Deployment 3',
        entityId: '22222',
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
    const initialExpandedRows = flows
        .filter((row) => row.children && !!row.children.length)
        .map((row) => row.id); // Default to all expanded
    const [expandedRows, setExpandedRows] = React.useState<string[]>(initialExpandedRows);
    const [selectedRows, setSelectedRows] = React.useState<string[]>([]);

    // derived data
    const numFlows = getNumFlows(flows);
    const allUniquePorts = getAllUniquePorts(flows);

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
                                <FlowsTableHeaderText type="active" numFlows={numFlows} />
                            </ToolbarItem>
                            <ToolbarItem alignment={{ default: 'alignRight' }}>
                                <FlowsBulkActions
                                    type="active"
                                    selectedRows={selectedRows}
                                    onClearSelectedRows={() => setSelectedRows([])}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <StackItem>
                    <FlowsTable
                        label="Deployment flows"
                        flows={flows}
                        numFlows={numFlows}
                        expandedRows={expandedRows}
                        setExpandedRows={setExpandedRows}
                        selectedRows={selectedRows}
                        setSelectedRows={setSelectedRows}
                        isEditable
                    />
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlow;
