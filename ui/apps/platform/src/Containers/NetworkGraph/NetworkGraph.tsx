/* eslint-disable @typescript-eslint/no-unsafe-return */
import React, { useMemo } from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { Popover } from '@patternfly/react-core';
import {
    SELECTION_EVENT,
    SelectionEventListener,
    useEventListener,
    TopologySideBar,
    TopologyView,
    createTopologyControlButtons,
    defaultControlButtonsOptions,
    TopologyControlBar,
    useVisualizationController,
    Visualization,
    VisualizationSurface,
    VisualizationProvider,
    Edge,
    Controller,
} from '@patternfly/react-topology';

import { networkBasePathPF } from 'routePaths';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';
import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';
import CidrBlockSideBar from './cidr/CidrBlockSideBar';
import ExternalEntitiesSideBar from './externalEntities/ExternalEntitiesSideBar';
import ExternalGroupSideBar from './external/ExternalGroupSideBar';
import NetworkPolicySimulatorSidePanel from './simulation/NetworkPolicySimulatorSidePanel';
import { getNodeById } from './utils/networkGraphUtils';
import { CustomModel, CustomNodeModel } from './types/topology.type';
import { Simulation } from './utils/getSimulation';
import LegendContent from './components/LegendContent';

import './Topology.css';
import useNetworkPolicySimulator, {
    ApplyNetworkPolicyModification,
    NetworkPolicySimulator,
    SetNetworkPolicyModification,
} from './hooks/useNetworkPolicySimulator';
import SimulationFrame from './simulation/SimulationFrame';

// TODO: move these type defs to a central location
export const UrlDetailType = {
    NAMESPACE: 'namespace',
    DEPLOYMENT: 'deployment',
    CIDR_BLOCK: 'cidr',
    EXTERNAL_ENTITIES: 'internet',
    EXTERNAL: 'external',
} as const;
export type UrlDetailTypeKey = keyof typeof UrlDetailType;
export type UrlDetailTypeValue = typeof UrlDetailType[UrlDetailTypeKey];

function getUrlParamsForEntity(type, id): [UrlDetailTypeValue, string] {
    return [UrlDetailType[type], id];
}

export type NetworkGraphProps = {
    model: CustomModel;
    simulation: Simulation;
    selectedNode?: CustomNodeModel;
    selectedClusterId: string;
    updateCount: number;
};

export type TopologyComponentProps = {
    model: CustomModel;
    simulation: Simulation;
    selectedClusterId: string;
    selectedNode?: CustomNodeModel;
    simulator: NetworkPolicySimulator;
    setNetworkPolicyModification: SetNetworkPolicyModification;
    applyNetworkPolicyModification: ApplyNetworkPolicyModification;
};

// @TODO: Consider a better approach to managing the side panel related state (simulation + URL path for entities)
function clearSimulationQuery(search: string): string {
    const modifiedSearchFilter = getQueryObject(search);
    delete modifiedSearchFilter.simulation;
    const queryString = getQueryString(modifiedSearchFilter);
    return queryString;
}

const TopologyComponent = ({
    model,
    simulation,
    selectedClusterId,
    selectedNode,
    simulator,
    setNetworkPolicyModification,
    applyNetworkPolicyModification,
}: TopologyComponentProps) => {
    const history = useHistory();
    const controller = useVisualizationController();
    console.log('TopologyComponent', selectedNode?.data);

    function closeSidebar() {
        const queryString = clearSimulationQuery(history.location.search);
        history.push(`${networkBasePathPF}${queryString}`);
    }

    function onSelect(ids: string[]) {
        console.log('TopologyComponent: onSelect');
        const newSelectedId = ids?.[0] || '';
        const newSelectedEntity = getNodeById(model?.nodes, newSelectedId);
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        if (newSelectedEntity) {
            const { data, id } = newSelectedEntity;
            const [newDetailType, newDetailId] = getUrlParamsForEntity(data.type, id);
            const queryString = clearSimulationQuery(history.location.search);
            // if found, and it's not the logical grouping of all external sources, then trigger URL update
            if (newDetailId !== 'EXTERNAL') {
                history.push(`${networkBasePathPF}/${newDetailType}/${newDetailId}${queryString}`);
            } else {
                // otherwise, return to the graph-only state
                history.push(`${networkBasePathPF}${queryString}`);
            }
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

    function resetViewCallback() {
        controller.getGraph().reset();
        controller.getGraph().layout();
    }

    React.useEffect(() => {
        console.log('TopologyComponent: useEffect [model]');
        controller.fromModel(model, true);
    }, [model]);

    useEventListener<SelectionEventListener>(SELECTION_EVENT, (ids) => {
        console.log('TopologyComponent: useEventListener');
        onSelect(ids);
    });

    const selectedIds = selectedNode ? [selectedNode.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={closeSidebar}>
                    {simulation.isOn && simulation.type === 'networkPolicy' && (
                        <NetworkPolicySimulatorSidePanel
                            selectedClusterId={selectedClusterId}
                            simulator={simulator}
                            setNetworkPolicyModification={setNetworkPolicyModification}
                            applyNetworkPolicyModification={applyNetworkPolicyModification}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'NAMESPACE' && (
                        <NamespaceSideBar
                            namespaceId={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'DEPLOYMENT' && (
                        <DeploymentSideBar
                            deploymentId={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'EXTERNAL_GROUP' && (
                        <ExternalGroupSideBar
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'CIDR_BLOCK' && (
                        <CidrBlockSideBar
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                    {selectedNode && selectedNode?.data?.type === 'EXTERNAL_ENTITIES' && (
                        <ExternalEntitiesSideBar
                            id={selectedNode.id}
                            nodes={model?.nodes || []}
                            edges={model?.edges || []}
                        />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedNode || simulation.isOn}
            sideBarResizable
            controlBar={
                <TopologyControlBar
                    controlButtons={createTopologyControlButtons({
                        ...defaultControlButtonsOptions,
                        zoomInCallback,
                        zoomOutCallback,
                        fitToScreenCallback,
                        resetViewCallback,
                    })}
                />
            }
        >
            <VisualizationSurface state={{ selectedIds }} />
            <Popover
                aria-label="Network graph legend"
                bodyContent={<LegendContent />}
                hasAutoWidth
                reference={() => document.getElementById('legend') as HTMLButtonElement}
            />
        </TopologyView>
    );
};

function compareModels(prevProps, nextProps) {
    console.log(
        'NetworkGraph: compareModels prevProps nextProps',
        prevProps.updateCount,
        nextProps.updateCount
    );
    return (
        prevProps.updateCount === nextProps.updateCount &&
        prevProps.simulation.isOn === nextProps.simulation.isOn &&
        prevProps.simulation.type === nextProps.simulation.type
    );
}

const NetworkGraph = React.memo<NetworkGraphProps>(
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    ({ model, simulation, selectedClusterId, selectedNode, updateCount }) => {
        const controller = useMemo(() => new Visualization(), []);
        controller.registerLayoutFactory(defaultLayoutFactory);
        controller.registerComponentFactory(defaultComponentFactory);
        controller.registerComponentFactory(stylesComponentFactory);

        const { simulator, setNetworkPolicyModification, applyNetworkPolicyModification } =
            useNetworkPolicySimulator({
                simulation,
                clusterId: selectedClusterId,
            });

        console.log('NetworkGraph');

        const isSimulating =
            simulator.state === 'GENERATED' ||
            simulator.state === 'UNDO' ||
            simulator.state === 'UPLOAD' ||
            (simulation.isOn && simulation.type === 'baseline');

        return (
            <SimulationFrame simulator={simulator}>
                <VisualizationProvider controller={controller}>
                    <TopologyComponent
                        model={model}
                        simulation={simulation}
                        selectedClusterId={selectedClusterId}
                        simulator={simulator}
                        selectedNode={selectedNode}
                        setNetworkPolicyModification={setNetworkPolicyModification}
                        applyNetworkPolicyModification={applyNetworkPolicyModification}
                    />
                </VisualizationProvider>
            </SimulationFrame>
        );
    },
    compareModels
);

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
