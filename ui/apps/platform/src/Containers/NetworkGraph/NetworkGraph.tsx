/* eslint-disable @typescript-eslint/no-unsafe-return */
import React from 'react';
import { useHistory } from 'react-router-dom';
import {
    Model,
    SELECTION_EVENT,
    TopologySideBar,
    TopologyView,
    useVisualizationController,
    Visualization,
    VisualizationProvider,
    VisualizationSurface,
} from '@patternfly/react-topology';

import { networkBasePathPF } from 'routePaths';
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

function findEntityById(
    graphModel: Model,
    id: string,
    type: UrlDetailTypeValue | undefined = undefined
): Record<string, any> | undefined {
    if (
        type === UrlDetailType.DEPLOYMENT ||
        type === UrlDetailType.CIDR_BLOCK ||
        type === UrlDetailType.EXTERNAL_ENTITIES
    ) {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        return graphModel.nodes?.find((node: { id: string }) => node.id === id);
    }
    if (type === UrlDetailType.NAMESPACE) {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        return graphModel.groups?.find((group: { id: string }) => group.id === id);
    }
    let entity;
    entity = graphModel.nodes?.find((node: { id: string }) => node.id === id);
    if (!entity) {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        entity = graphModel.groups?.find((group: { id: string }) => group.id === id);
    }
    return entity;
}

function getUrlParamsForEntity(selectEntity: {
    data: { entityType: string | number };
    id: string;
}): [UrlDetailTypeValue, string] {
    const detailType = UrlDetailType[selectEntity.data.entityType];
    const detailId = selectEntity.id;

    return [detailType, detailId];
}

export type NetworkGraphProps = {
    detailType?: UrlDetailTypeValue;
    detailId?: string;
    model: Model;
};

export type TopologyComponentProps = NetworkGraphProps;

const TopologyComponent = ({ detailType, detailId, model }: TopologyComponentProps) => {
    const selectedEntity = detailId && findEntityById(model, detailId, detailType);
    const history = useHistory();

    const controller = useVisualizationController();

    React.useEffect(() => {
        function onSelect(ids: string[]) {
            const newSelectedId = ids?.[0] || null;
            // try to find the selected ID in the various types of graph objects: nodes and groups
            const newSelectedEntity = newSelectedId && findEntityById(model, newSelectedId);
            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
            // @ts-ignore
            const [newDetailType, newDetailId] = getUrlParamsForEntity(newSelectedEntity);

            // if found, and it's not the logical grouping of all external sources, then trigger URL update
            if (newSelectedEntity && newSelectedEntity?.data?.entityType !== 'EXTERNAL') {
                history.push(`${networkBasePathPF}/${newDetailType}/${newDetailId}`);
            } else {
                // otherwise, return to the graph-only state
                history.push(`${networkBasePathPF}`);
            }
        }

        controller.fromModel(model, false);
        controller.addEventListener(SELECTION_EVENT, onSelect);

        return () => {
            controller.removeEventListener(SELECTION_EVENT, onSelect);
        };
    }, [controller, history]);

    function closeSidebar() {
        history.push(`${networkBasePathPF}`);
    }

    const selectedIds = selectedEntity ? [selectedEntity.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={closeSidebar}>
                    {selectedEntity && selectedEntity?.data?.entityType === 'NAMESPACE' && (
                        <NamespaceSideBar />
                    )}
                    {selectedEntity && selectedEntity?.data?.entityType === 'DEPLOYMENT' && (
                        <DeploymentSideBar />
                    )}
                    {selectedEntity && selectedEntity?.data?.entityType === 'CIDR_BLOCK' && (
                        <CidrBlockSideBar />
                    )}
                    {selectedEntity && selectedEntity?.data?.entityType === 'EXTERNAL_ENTITIES' && (
                        <ExternalEntitiesSideBar />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedEntity}
            sideBarResizable
        >
            <VisualizationSurface state={{ selectedIds }} />
        </TopologyView>
    );
};

const NetworkGraph = React.memo<NetworkGraphProps>(({ detailType, detailId, model }) => {
    const controller = new Visualization();
    controller.registerLayoutFactory(defaultLayoutFactory);
    controller.registerComponentFactory(defaultComponentFactory);
    controller.registerComponentFactory(stylesComponentFactory);

    return (
        <div className="pf-ri__topology-demo">
            <VisualizationProvider controller={controller}>
                <TopologyComponent detailType={detailType} detailId={detailId} model={model} />
            </VisualizationProvider>
        </div>
    );
});

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
