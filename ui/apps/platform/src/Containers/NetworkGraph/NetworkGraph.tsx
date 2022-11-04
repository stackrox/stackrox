/* eslint-disable @typescript-eslint/no-unsafe-return */
import React from 'react';
import {
    Model,
    SELECTION_EVENT,
    TopologySideBar,
    TopologyView,
    createTopologyControlButtons,
    defaultControlButtonsOptions,
    TopologyControlBar,
    useVisualizationController,
    Visualization,
    VisualizationProvider,
    VisualizationSurface,
    NodeModel,
} from '@patternfly/react-topology';

import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';
import CidrBlockSideBar from './cidr/CidrBlockSideBar';
import ExternalEntitiesSideBar from './external/ExternalEntitiesSideBar';

import './Topology.css';

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

function findEntityById(graphModel: Model, id: string | undefined): NodeModel | undefined {
    const entity = graphModel.nodes?.find((node: { id: string }) => node.id === id);
    return entity;
}

function getUrlParamsForEntity(selectedEntity: NodeModel): [UrlDetailTypeValue, string] {
    const detailType = UrlDetailType[selectedEntity.data.type];
    const detailId = selectedEntity.id;

    return [detailType, detailId];
}

export type NetworkGraphProps = {
    detailType?: UrlDetailTypeValue;
    detailId?: string;
    model: Model;
    closeSidebar: () => void;
    onSelectNode: (type, id) => void;
};

export type TopologyComponentProps = NetworkGraphProps;

const TopologyComponent = ({
    detailId,
    model,
    closeSidebar,
    onSelectNode,
}: TopologyComponentProps) => {
    const selectedEntity = findEntityById(model, detailId);

    const controller = useVisualizationController();

    React.useEffect(() => {
        function onSelect(ids: string[]) {
            const newSelectedId = ids?.[0] || '';
            const newSelectedEntity = findEntityById(model, newSelectedId);
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            if (newSelectedEntity) {
                const [newDetailType, newDetailId] = getUrlParamsForEntity(newSelectedEntity);
                onSelectNode(newDetailType, newDetailId);
            }
        }

        controller.fromModel(model, false);
        controller.addEventListener(SELECTION_EVENT, onSelect);

        return () => {
            controller.removeEventListener(SELECTION_EVENT, onSelect);
        };
    }, [controller, model, onSelectNode]);

    const selectedIds = selectedEntity ? [selectedEntity.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={closeSidebar}>
                    {selectedEntity && selectedEntity?.data?.type === 'NAMESPACE' && (
                        <NamespaceSideBar />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'DEPLOYMENT' && (
                        <DeploymentSideBar />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'CIDR_BLOCK' && (
                        <CidrBlockSideBar />
                    )}
                    {selectedEntity && selectedEntity?.data?.type === 'EXTERNAL_ENTITIES' && (
                        <ExternalEntitiesSideBar />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedEntity}
            sideBarResizable
            controlBar={
                <TopologyControlBar
                    controlButtons={createTopologyControlButtons({
                        ...defaultControlButtonsOptions,
                        zoomInCallback: () => {
                            controller.getGraph().scaleBy(4 / 3);
                        },
                        zoomOutCallback: () => {
                            controller.getGraph().scaleBy(0.75);
                        },
                        fitToScreenCallback: () => {
                            controller.getGraph().fit(80);
                        },
                        resetViewCallback: () => {
                            controller.getGraph().reset();
                            controller.getGraph().layout();
                        },
                        legendCallback: () => {
                            // console.log('hi');
                        },
                    })}
                />
            }
        >
            <VisualizationSurface state={{ selectedIds }} />
        </TopologyView>
    );
};

const NetworkGraph = React.memo<NetworkGraphProps>(
    ({ detailType, detailId, closeSidebar, onSelectNode, model }) => {
        const controller = new Visualization();
        controller.registerLayoutFactory(defaultLayoutFactory);
        controller.registerComponentFactory(defaultComponentFactory);
        controller.registerComponentFactory(stylesComponentFactory);

        return (
            <div className="pf-ri__topology-demo">
                <VisualizationProvider controller={controller}>
                    <TopologyComponent
                        detailType={detailType}
                        detailId={detailId}
                        model={model}
                        closeSidebar={closeSidebar}
                        onSelectNode={onSelectNode}
                    />
                </VisualizationProvider>
            </div>
        );
    }
);

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
