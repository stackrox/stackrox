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

import BulkActionsDropdown from 'Components/PatternFly/BulkActionsDropdown';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';

import './DeploymentFlows.css';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import EntityNameSearchInput from '../common/EntityNameSearchInput';
import { Flow } from '../types';
import { getAllUniquePorts, getNumFlows } from '../utils/flowUtils';
import FlowsTable from '../common/FlowsTable';

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
    const numFlows = getNumFlows(flows);
    const allUniquePorts = getAllUniquePorts(flows);

    // setter functions
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
                                    <Text component={TextVariants.h3}>{numFlows} active flows</Text>
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
                    <FlowsTable
                        label="Deployment flows"
                        flows={flows}
                        numFlows={numFlows}
                        expandedRows={expandedRows}
                        setExpandedRows={setExpandedRows}
                        selectedRows={selectedRows}
                        setSelectedRows={setSelectedRows}
                    />
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentFlow;
