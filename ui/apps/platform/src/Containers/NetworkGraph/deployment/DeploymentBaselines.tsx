import React from 'react';
import {
    Button,
    Checkbox,
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Switch,
    Tooltip,
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Switch,
    Tooltip,
} from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

import EntityNameSearchInput from '../common/EntityNameSearchInput';
import AdvancedFlowsFilter, {
    defaultAdvancedFlowsFilters,
} from '../common/AdvancedFlowsFilter/AdvancedFlowsFilter';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { Flow } from '../types';
import { getAllUniqPorts } from '../utils/flowUtils';

const baselines: Flow[] = [
    {
        id: 'External Entities-Ingress-Many-TCP',
        type: 'External',
        entity: 'External Entities',
        namespace: '',
        direction: 'Ingress',
        port: 'Many',
        protocol: 'TCP',
        isAnomalous: false,
        children: [
            {
                id: 'External Entities-Ingress-443-TCP',
                type: 'External',
                entity: 'External Entities',
                namespace: '',
                direction: 'Ingress',
                port: '443',
                protocol: 'TCP',
                isAnomalous: false,
            },
            {
                id: 'External Entities-Ingress-9443-TCP',
                type: 'External',
                entity: 'External Entities',
                namespace: '',
                direction: 'Ingress',
                port: '9443',
                protocol: 'TCP',
                isAnomalous: false,
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
        isAnomalous: false,
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

function DeploymentBaselines() {
    const [isAlertingOnViolations, setIsAlertingOnViolations] = React.useState<boolean>(false);
    const [isExcludingPortsAndProtocols, setIsExcludingPortsAndProtocols] =
        React.useState<boolean>(false);
    const [entityNameFilter, setEntityNameFilter] = React.useState<string>('');
    const [advancedFilters, setAdvancedFilters] = React.useState<AdvancedFlowsFilterType>(
        defaultAdvancedFlowsFilters
    );

    const allUniqPorts = getAllUniqPorts(baselines);

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem>
                            <Switch
                                id="simple-switch"
                                label="Alert on baseline violation"
                                isChecked={isAlertingOnViolations}
                                onChange={setIsAlertingOnViolations}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Tooltip
                                content={
                                    <div>
                                        Trigger violations for network policies not in the baseline
                                    </div>
                                }
                            >
                                <HelpIcon className="pf-u-color-200" />
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                </StackItem>
                <Divider component="hr" />
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
                                allUniqPorts={allUniqPorts}
                            />
                        </FlexItem>
                    </Flex>
                </StackItem>
                <Divider component="hr" />
                <StackItem isFilled>@TODO: Table</StackItem>
                <Divider component="hr" />
                <StackItem>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                        justifyContent={{ default: 'justifyContentCenter' }}
                    >
                        <FlexItem>
                            <Checkbox
                                id="exclude-ports-and-protocols-checkbox"
                                label="Exclude ports & protocols"
                                isChecked={isExcludingPortsAndProtocols}
                                onChange={setIsExcludingPortsAndProtocols}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Button variant="primary">Simulate baseline as network policy</Button>
                        </FlexItem>
                    </Flex>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselines;
