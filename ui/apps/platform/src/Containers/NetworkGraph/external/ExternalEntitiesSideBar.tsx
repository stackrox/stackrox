import React, { ReactElement, useEffect, useState } from 'react';
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

import { TimeWindow } from 'constants/timeWindows';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { UseUrlSearchReturn } from 'hooks/useURLSearch';

import { ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import ExternalFlowsTable from './ExternalFlowsTable';
import ExternalIpsContainer from './ExternalIpsContainer';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import { getNodeById } from '../utils/networkGraphUtils';
import EntityDetails from './EntityDetails';

export type ExternalEntitiesView = 'external-ips' | 'workloads-with-external-flows';

export type ExternalEntitiesSideBarProps = {
    labelledById: string;
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    selectedExternalIP: string | string[] | undefined;
    scopeHierarchy: NetworkScopeHierarchy;
    onNodeSelect: (id: string) => void;
    onExternalIPSelect: (externalIP: string | undefined) => void;
    timeWindow: TimeWindow;
    urlPagination: UseURLPaginationResult;
    urlSearchFiltering: UseUrlSearchReturn;
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
    selectedExternalIP,
    onNodeSelect,
    onExternalIPSelect,
    timeWindow,
    urlPagination,
    urlSearchFiltering,
}: ExternalEntitiesSideBarProps): ReactElement {
    const [selectedView, setSelectedView] = useState<ExternalEntitiesView>('external-ips');

    const entityNode = getNodeById(nodes, id);
    const { setPage } = urlPagination;
    const { setSearchFilter } = urlSearchFiltering;

    useEffect(() => {
        setPage(1);
        setSearchFilter({});
    }, [selectedExternalIP, selectedView, setPage, setSearchFilter]);

    if (selectedExternalIP) {
        return (
            <EntityDetails
                labelledById={labelledById}
                entityName={entityNode?.label || ''}
                entityId={String(selectedExternalIP)}
                scopeHierarchy={scopeHierarchy}
                onNodeSelect={onNodeSelect}
                onExternalIPSelect={onExternalIPSelect}
                timeWindow={timeWindow}
                urlPagination={urlPagination}
                urlSearchFiltering={urlSearchFiltering}
            />
        );
    }

    return (
        <Stack>
            <StackItem>
                <Flex direction={{ default: 'row' }} className="pf-v5-u-p-md pf-v5-u-mb-0">
                    <FlexItem>
                        <ExternalEntitiesIcon />
                    </FlexItem>
                    <FlexItem>
                        <EntityTitleText text={entityNode?.label} id={labelledById} />
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
                        <ExternalIpsContainer
                            scopeHierarchy={scopeHierarchy}
                            onExternalIPSelect={onExternalIPSelect}
                            timeWindow={timeWindow}
                            urlPagination={urlPagination}
                            urlSearchFiltering={urlSearchFiltering}
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
