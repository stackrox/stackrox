import React, { useEffect, useCallback, useRef } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom-v5-compat';
import { Popover } from '@patternfly/react-core';
import {
    SELECTION_EVENT,
    useEventListener,
    TopologySideBar,
    TopologyView,
    createTopologyControlButtons,
    defaultControlButtonsOptions,
    TopologyControlBar,
    useVisualizationController,
    VisualizationSurface,
} from '@patternfly/react-topology';
import type { SelectionEventListener } from '@patternfly/react-topology';

import { networkBasePath } from 'routePaths';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import useFetchDeploymentCount from 'hooks/useFetchDeploymentCount';
import { getQueryString } from 'utils/queryStringUtils';
import type { QueryValue } from 'hooks/useURLParameter';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';
import GenericEntitiesSideBar from './genericEntities/GenericEntitiesSideBar';
import ExternalEntitiesSideBar from './external/ExternalEntitiesSideBar';
import ExternalGroupSideBar from './external/ExternalGroupSideBar';
import NetworkPolicySimulatorSidePanel, {
    clearSidePanelQuery,
} from './simulation/NetworkPolicySimulatorSidePanel';
import { getExternalEntitiesNode, getNodeById } from './utils/networkGraphUtils';
import { isNodeOfType } from './types/topology.type';
import type { CustomModel, CustomNodeModel } from './types/topology.type';
import type { Simulation } from './utils/getSimulation';
import LegendContent from './components/LegendContent';

import type { EdgeState } from './components/EdgeStateSelect';
import EmptyUnscopedState from './components/EmptyUnscopedState';
import type {
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from './hooks/useNetworkPolicySimulator';
import type { NetworkScopeHierarchy } from './types/networkScopeHierarchy';
import { getSearchFilterFromScopeHierarchy } from './utils/simulatorUtils';
import {
    CidrBlockIcon,
    ExternalEntitiesIcon,
    InternalEntitiesIcon,
} from './common/NetworkGraphIcons';
import { DEFAULT_NETWORK_GRAPH_PAGE_SIZE } from './NetworkGraph.constants';

import {
    usePagination,
    usePaginationSecondary,
    useSearchFilterSidePanel,
    useSidePanelTab,
    useSidePanelToggle,
} from './NetworkGraphURLStateContext';

// TODO: move these type defs to a central location
export const UrlNodeType = {
    NAMESPACE: 'namespace',
    DEPLOYMENT: 'deployment',
    CIDR_BLOCK: 'cidr',
    EXTERNAL_ENTITIES: 'internet',
    EXTERNAL_GROUP: 'external',
    INTERNAL_ENTITIES: 'internal',
} as const;
export type UrlNodeTypeKey = keyof typeof UrlNodeType;
export type UrlNodeTypeValue = (typeof UrlNodeType)[UrlNodeTypeKey];

function getUrlParamsForNode(type, id): [UrlNodeTypeValue, string] {
    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return [UrlNodeType[type], id];
}

export type TopologyComponentProps = {
    isReadyForVisualization: boolean;
    model: CustomModel;
    simulation: Simulation;
    selectedNode?: CustomNodeModel;
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
    edgeState: EdgeState;
    scopeHierarchy: NetworkScopeHierarchy;
};

const TopologyComponent = ({
    isReadyForVisualization,
    model,
    simulation,
    selectedNode,
    simulator,
    setNetworkPolicyModification,
    edgeState,
    scopeHierarchy,
}: TopologyComponentProps) => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNetworkGraphExternalIpsEnabled = isFeatureFlagEnabled('ROX_NETWORK_GRAPH_EXTERNAL_IPS');

    const { hasReadAccess } = usePermissions();
    const hasReadAccessForNetworkPolicy = hasReadAccess('NetworkPolicy');

    const { detailID: selectedExternalIP, nodeId: nodeIdParam } = useParams();

    // two paginations (default + secondary for when 2 tables present)
    // deployment flows section has 2 tables instead of 1 which is why we need the additional pagination.
    const { setPerPage } = usePagination();
    const { setPerPage: setPerPageSecondary } = usePaginationSecondary();

    const { setSelectedTabSidePanel } = useSidePanelTab();
    const { setSelectedToggleSidePanel } = useSidePanelToggle();

    const { setSearchFilter } = useSearchFilterSidePanel();

    const firstRenderRef = useRef(true);
    const location = useLocation();
    const navigate = useNavigate();
    const controller = useVisualizationController();

    const closeSidebar = useCallback(() => {
        const queryObject = clearSidePanelQuery(location.search);
        const queryString = getQueryString(queryObject);
        navigate(`${networkBasePath}${queryString}`);
    }, [navigate, location.search]);

    type OnNavigateArgs = {
        nodeID: string;
        externalIP?: string;
        parametersQuery?: QueryValue;
        searchFilter?: QueryValue;
    };

    function onNavigate({ nodeID, externalIP, parametersQuery, searchFilter }: OnNavigateArgs) {
        const newSelectedId = nodeID || '';
        const newSelectedEntity = getNodeById(model?.nodes, newSelectedId);

        if (selectedNode && !newSelectedId) {
            closeSidebar();
            return;
        }
        if (!newSelectedEntity) {
            return;
        }

        const queryObject = clearSidePanelQuery(location.search);
        if (parametersQuery) {
            Object.assign(queryObject, parametersQuery);
        }

        if (searchFilter) {
            Object.assign(queryObject, searchFilter);
        }

        const queryString = getQueryString(queryObject);

        const { data, id } = newSelectedEntity;
        const [nodeType, nodeId] = getUrlParamsForNode(data.type, id);

        if (nodeId !== 'EXTERNAL') {
            let url = `${networkBasePath}/${nodeType}/${encodeURIComponent(nodeId)}`;
            if (externalIP) {
                url += `/externalIP/${externalIP}`;
            }
            navigate(`${url}${queryString}`);
        } else {
            navigate(`${networkBasePath}${queryString}`);
        }
    }

    const { deploymentCount } = useFetchDeploymentCount(
        getSearchFilterFromScopeHierarchy(scopeHierarchy)
    );

    function onNodeSelect(nodeID: string, parametersQuery?: QueryValue, searchFilter?: QueryValue) {
        onNavigate({ nodeID, parametersQuery, searchFilter });
    }

    function onExternalIPSelect(externalIP: string | undefined) {
        const externalEntitiesNode = getExternalEntitiesNode(model.nodes);
        if (externalEntitiesNode) {
            onNavigate({ nodeID: externalEntitiesNode.id, externalIP });
        }
    }

    function zoomInCallback() {
        controller.getGraph().scaleBy(4 / 3);
    }

    function zoomOutCallback() {
        controller.getGraph().scaleBy(0.75);
    }

    function fitToScreenCallback() {
        controller.getGraph().fit(80);
    }

    const resetViewCallback = useCallback(() => {
        const graph = controller.getGraph();
        graph.reset();
        // only layout if there are nodes to layout, otherwise it will throw an error
        if (graph.getNodes().length > 0) {
            graph.layout();
        }
    }, [controller]);

    const panNodeIntoView = useCallback(
        (node: CustomNodeModel) => {
            const selectedNodeElement = controller.getNodeById(node.id);
            if (selectedNodeElement) {
                // the offset is to make sure the label also makes it inside the viewport
                controller.getGraph().panIntoView(selectedNodeElement, { offset: 50 });
            }
        },
        [controller]
    );

    useEventListener<SelectionEventListener>(SELECTION_EVENT, (ids) => {
        onNavigate({ nodeID: ids?.[0] || '' });
    });

    useEffect(() => {
        if (!nodeIdParam) {
            setPerPage(DEFAULT_NETWORK_GRAPH_PAGE_SIZE, 'replace');
            setPerPageSecondary(DEFAULT_NETWORK_GRAPH_PAGE_SIZE, 'replace');
            setSearchFilter({}, 'replace');
            setSelectedTabSidePanel(undefined, 'replace');
            setSelectedToggleSidePanel(undefined, 'replace');
        }
    }, [
        setPerPage,
        setSearchFilter,
        setPerPageSecondary,
        setSelectedTabSidePanel,
        setSelectedToggleSidePanel,
        nodeIdParam,
    ]);

    useEffect(() => {
        // we don't want to reset view on init
        if (!firstRenderRef.current && controller.hasGraph()) {
            resetViewCallback();
        } else {
            firstRenderRef.current = false;
        }
    }, [controller, edgeState, resetViewCallback]);

    useEffect(() => {
        controller.fromModel(model);
        if (selectedNode) {
            panNodeIntoView(selectedNode);
        } else if (
            location.pathname !== networkBasePath &&
            !selectedNode &&
            model.nodes.length > 0
        ) {
            // If there's no selected node but the user is on a node-specific URL (and we've
            // confirmed nodes have been fetched), reset to the base path by closing the sidebar.
            // This also handles the edge case where a user might land on a node URL before node data
            // is available – we want to prevent closing the sidebar until data has been fetched
            closeSidebar();
        }
    }, [controller, location.pathname, model, selectedNode, closeSidebar, panNodeIntoView]);

    const selectedIds = selectedNode ? [selectedNode.id] : [];

    const labelledById = 'TopologySideBarLabelledBy';

    return (
        <TopologyView
            sideBar={
                <TopologySideBar aria-labelledby={labelledById} resizable onClose={closeSidebar}>
                    {hasReadAccessForNetworkPolicy &&
                        simulation.isOn &&
                        simulation.type === 'networkPolicy' && (
                            <NetworkPolicySimulatorSidePanel
                                labelledById={labelledById}
                                simulator={simulator}
                                setNetworkPolicyModification={setNetworkPolicyModification}
                                scopeHierarchy={scopeHierarchy}
                                scopeDeploymentCount={deploymentCount ?? 0}
                            />
                        )}
                    {selectedNode && selectedNode?.data?.type === 'NAMESPACE' && (
                        <NamespaceSideBar
                            labelledById={labelledById}
                            namespaceId={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'DEPLOYMENT' && (
                        <DeploymentSideBar
                            labelledById={labelledById}
                            deploymentId={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            edgeState={edgeState}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'EXTERNAL_GROUP' && (
                        <ExternalGroupSideBar
                            labelledById={labelledById}
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                        />
                    )}
                    {selectedNode && isNodeOfType('CIDR_BLOCK', selectedNode) && (
                        <GenericEntitiesSideBar
                            labelledById={labelledById}
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                            EntityHeaderIcon={<CidrBlockIcon />}
                            sidebarTitle={selectedNode.data.externalSource.cidr ?? ''}
                            flowTableLabel="Cidr block flows"
                        />
                    )}
                    {selectedNode &&
                        isNodeOfType('EXTERNAL_ENTITIES', selectedNode) &&
                        (isNetworkGraphExternalIpsEnabled ? (
                            <ExternalEntitiesSideBar
                                labelledById={labelledById}
                                id={selectedNode.id}
                                nodes={model?.nodes || []}
                                edges={model?.edges || []}
                                scopeHierarchy={scopeHierarchy}
                                selectedExternalIP={selectedExternalIP}
                                onNodeSelect={onNodeSelect}
                                onExternalIPSelect={onExternalIPSelect}
                            />
                        ) : (
                            <GenericEntitiesSideBar
                                labelledById={labelledById}
                                id={selectedNode.id}
                                nodes={model?.nodes || []}
                                edges={model?.edges || []}
                                onNodeSelect={onNodeSelect}
                                EntityHeaderIcon={<ExternalEntitiesIcon />}
                                sidebarTitle={'Connected entities outside your cluster'}
                                flowTableLabel="External entities flows"
                            />
                        ))}
                    {selectedNode && isNodeOfType('INTERNAL_ENTITIES', selectedNode) && (
                        <GenericEntitiesSideBar
                            labelledById={labelledById}
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                            onNodeSelect={onNodeSelect}
                            EntityHeaderIcon={<InternalEntitiesIcon />}
                            sidebarTitle={'Unknown entity connections within your clusters'}
                            flowTableLabel="Internal entities flows"
                        />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedNode || simulation.isOn}
            sideBarResizable
            controlBar={
                isReadyForVisualization ? (
                    <TopologyControlBar
                        controlButtons={createTopologyControlButtons({
                            ...defaultControlButtonsOptions,
                            zoomInCallback,
                            zoomOutCallback,
                            fitToScreenCallback,
                            resetViewCallback,
                        })}
                    />
                ) : undefined
            }
        >
            {isReadyForVisualization ? (
                <>
                    <VisualizationSurface state={{ selectedIds }} />
                    <Popover
                        aria-label="Network graph legend"
                        bodyContent={<LegendContent />}
                        hasAutoWidth
                        triggerRef={() => document.getElementById('legend') as HTMLButtonElement}
                    />
                </>
            ) : (
                <div className="pf-v5-u-h-100 pf-v5-u-w-100 pf-v5-u-background-color-100">
                    <EmptyUnscopedState />
                </div>
            )}
        </TopologyView>
    );
};

export default TopologyComponent;
