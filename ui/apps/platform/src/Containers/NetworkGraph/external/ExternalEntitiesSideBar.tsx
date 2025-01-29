import React, { ReactElement, useState } from 'react';
import {
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Text,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import { ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import ExternalFlowsTable from './ExternalFlowsTable';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import { getNodeById } from '../utils/networkGraphUtils';

export type ExternalEntitiesView = 'external-ips' | 'workloads-with-external-flows';

export type ExternalEntitiesSideBarProps = {
    labelledById: string;
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    onNodeSelect: (id: string) => void;
};

function ExternalEntitiesSideBar({
    labelledById,
    id,
    nodes,
    edges,
    onNodeSelect,
}: ExternalEntitiesSideBarProps): ReactElement {
    const [selectedView, setSelectedView] = useState<ExternalEntitiesView>('external-ips');
    const entityNode = getNodeById(nodes, id);

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <ExternalEntitiesIcon />
                    </FlexItem>
                    <FlexItem>
                        <Title headingLevel="h2" id={labelledById}>
                            {entityNode?.label}
                        </Title>
                        <Text className="pf-v5-u-font-size-sm pf-v5-u-color-200">
                            Connected entities outside your cluster
                        </Text>
                    </FlexItem>
                </Flex>
            </StackItem>
            <Divider component="hr" />
            <StackItem className="pf-v5-u-p-md">
                <ToggleGroup aria-label="Toggle between external IPs and workload flows view">
                    <ToggleGroupItem
                        text="External IPs"
                        buttonId="external-ips"
                        isSelected={selectedView === 'external-ips'}
                        onChange={() => setSelectedView('external-ips')}
                    />
                    <ToggleGroupItem
                        text="Workloads with external flows"
                        buttonId="workloads-with-external-flows"
                        isSelected={selectedView === 'workloads-with-external-flows'}
                        onChange={() => setSelectedView('workloads-with-external-flows')}
                    />
                </ToggleGroup>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <Stack className="pf-v5-u-p-md">
                    {selectedView === 'external-ips' ? (
                        <div>external ips</div>
                    ) : (
                        <ExternalFlowsTable
                            nodes={nodes}
                            edges={edges}
                            id={id}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                </Stack>
            </StackItem>
        </Stack>
    );
}

export default ExternalEntitiesSideBar;
