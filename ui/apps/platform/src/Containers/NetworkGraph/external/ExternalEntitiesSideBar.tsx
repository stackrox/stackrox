/* eslint-disable no-nested-ternary */
import React, { ReactElement, useState } from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
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

import { ExternalSourceNetworkEntityInfo } from 'types/networkFlow.proto';

import { ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import EntityDetailsTable from './EntityDetailsTable';
import ExternalFlowsTable from './ExternalFlowsTable';
import ExternalIpsTable from './ExternalIpsTable';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import { getNodeById } from '../utils/networkGraphUtils';

export type ExternalEntitiesView = 'external-ips' | 'workloads-with-external-flows';

export type ExternalEntitiesSideBarProps = {
    labelledById: string;
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    scopeHierarchy: NetworkScopeHierarchy;
    onNodeSelect: (id: string) => void;
};

function EntityTitleText({ text, id }: { text: string | undefined; id: string }) {
    return (
        <Title headingLevel="h2" id={id}>
            {text}
        </Title>
    );
}

function ExternalEntitiesSideBar({
    labelledById,
    id,
    nodes,
    edges,
    scopeHierarchy,
    onNodeSelect,
}: ExternalEntitiesSideBarProps): ReactElement {
    const [selectedView, setSelectedView] = useState<ExternalEntitiesView>('external-ips');
    const [selectedEntity, setSelectedEntity] = useState<ExternalSourceNetworkEntityInfo | null>(
        null
    );
    const entityNode = getNodeById(nodes, id);

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <ExternalEntitiesIcon />
                    </FlexItem>
                    <FlexItem>
                        {selectedEntity ? (
                            <Breadcrumb>
                                <BreadcrumbItem to="#" onClick={() => setSelectedEntity(null)}>
                                    <EntityTitleText text={entityNode?.label} id={labelledById} />
                                </BreadcrumbItem>
                                <BreadcrumbItem isActive>
                                    <EntityTitleText
                                        text={selectedEntity.externalSource.name}
                                        id={selectedEntity.externalSource.name}
                                    />
                                </BreadcrumbItem>
                            </Breadcrumb>
                        ) : (
                            <EntityTitleText text={entityNode?.label} id={labelledById} />
                        )}
                        <Text className="pf-v5-u-font-size-sm pf-v5-u-color-200">
                            Connected entities outside your cluster
                        </Text>
                    </FlexItem>
                </Flex>
            </StackItem>
            {!selectedEntity && (
                <>
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
                </>
            )}
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <Stack className="pf-v5-u-p-md">
                    {selectedEntity ? (
                        <EntityDetailsTable
                            entityId={selectedEntity.id}
                            scopeHierarchy={scopeHierarchy}
                            onNodeSelect={onNodeSelect}
                        />
                    ) : selectedView === 'external-ips' ? (
                        <ExternalIpsTable
                            scopeHierarchy={scopeHierarchy}
                            setSelectedEntity={setSelectedEntity}
                        />
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
