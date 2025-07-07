import React, { ReactElement } from 'react';
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

import { QueryValue } from 'hooks/useURLParameter';

import { ExternalEntitiesIcon } from '../common/NetworkGraphIcons';
import ExternalFlowsTable from './ExternalFlowsTable';
import ExternalIpsContainer from './ExternalIpsContainer';
import { NetworkScopeHierarchy } from '../types/networkScopeHierarchy';
import { CustomEdgeModel, CustomNodeModel } from '../types/topology.type';
import { getNodeById } from '../utils/networkGraphUtils';
import EntityDetails from './EntityDetails';

import {
    usePagination,
    useSearchFilterSidePanel,
    useSidePanelToggle,
} from '../NetworkGraphURLStateContext';

const EXTERNAL_ENTITIES_TOGGLES = ['EXTERNAL_IPS', 'WORKLOAD_EXTERNAL_FLOWS'] as const;
export type ExternalEntitiesToggleKey = (typeof EXTERNAL_ENTITIES_TOGGLES)[number];

export const DEFAULT_EXTERNAL_ENTITIES_TOGGLE: ExternalEntitiesToggleKey = 'EXTERNAL_IPS';

export function isValidExternalEntitiesToggle(
    value: QueryValue
): value is ExternalEntitiesToggleKey {
    return typeof value === 'string' && EXTERNAL_ENTITIES_TOGGLES.some((state) => state === value);
}

export type ExternalEntitiesSideBarProps = {
    labelledById: string;
    id: string;
    nodes: CustomNodeModel[];
    edges: CustomEdgeModel[];
    selectedExternalIP: string | string[] | undefined;
    scopeHierarchy: NetworkScopeHierarchy;
    onNodeSelect: (id: string) => void;
    onExternalIPSelect: (externalIP: string | undefined) => void;
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
}: ExternalEntitiesSideBarProps): ReactElement {
    const { selectedToggleSidePanel, setSelectedToggleSidePanel } = useSidePanelToggle();

    const entityNode = getNodeById(nodes, id);
    const { setPage } = usePagination();
    const { setSearchFilter } = useSearchFilterSidePanel();

    if (selectedExternalIP) {
        return (
            <EntityDetails
                labelledById={labelledById}
                entityName={entityNode?.label || ''}
                entityId={String(selectedExternalIP)}
                scopeHierarchy={scopeHierarchy}
                onNodeSelect={onNodeSelect}
                onExternalIPSelect={onExternalIPSelect}
            />
        );
    }

    const selectedView: ExternalEntitiesToggleKey = isValidExternalEntitiesToggle(
        selectedToggleSidePanel
    )
        ? selectedToggleSidePanel
        : DEFAULT_EXTERNAL_ENTITIES_TOGGLE;

    function handleToggle(view: ExternalEntitiesToggleKey) {
        setSelectedToggleSidePanel(view);
        setPage(1);
        setSearchFilter({});
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
                        buttonId="EXTERNAL_IPS"
                        isSelected={selectedView === 'EXTERNAL_IPS'}
                        onChange={() => handleToggle('EXTERNAL_IPS')}
                    />
                    <ToggleGroupItem
                        text="Workloads with external flows"
                        buttonId="WORKLOAD_EXTERNAL_FLOWS"
                        isSelected={selectedView === 'WORKLOAD_EXTERNAL_FLOWS'}
                        onChange={() => handleToggle('WORKLOAD_EXTERNAL_FLOWS')}
                    />
                </ToggleGroup>
            </StackItem>
            <Divider component="hr" />
            <StackItem isFilled style={{ overflow: 'auto' }}>
                <Stack className="pf-v5-u-p-md">
                    {selectedView === 'EXTERNAL_IPS' ? (
                        <ExternalIpsContainer
                            scopeHierarchy={scopeHierarchy}
                            onExternalIPSelect={onExternalIPSelect}
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
