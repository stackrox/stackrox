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

// TODO: replace this dummy data with real parsed graph data
const model: Model = {
    graph: {
        id: 'g1',
        type: 'graph',
        layout: 'ColaNoForce',
    },
    nodes: [
        {
            id: 'e337f873-64d8-46be-84ed-2a0c38c75fac',
            label: 'Central',
            type: 'node',
            width: 75,
            height: 75,
            data: {
                id: 'e337f873-64d8-46be-84ed-2a0c38c75fac',
                label: 'Central',
                type: 'node',
                width: 75,
                height: 75,
                entityType: 'DEPLOYMENT',
            },
        },
        {
            id: '09134b5d-8c12-41e8-821b-c97a5a1331c9',
            label: 'Sensor',
            type: 'node',
            width: 75,
            height: 75,
            data: {
                id: '09134b5d-8c12-41e8-821b-c97a5a1331c9',
                label: 'Sensor',
                type: 'node',
                width: 75,
                height: 75,
                entityType: 'DEPLOYMENT',
            },
        },
        {
            id: '__MzQuMTIwLjAuMC8xNg',
            label: 'Google/global | 34.120.0.0/16',
            type: 'node',
            width: 75,
            height: 75,
            data: {
                id: '__MzQuMTIwLjAuMC8xNg',
                label: 'Google/global | 34.120.0.0/16',
                type: 'node',
                width: 75,
                height: 75,
                entityType: 'CIDR_BLOCK',
            },
        },
        {
            id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
            label: 'External entities',
            type: 'node',
            width: 75,
            height: 75,
            data: {
                id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                label: 'External entities',
                type: 'node',
                width: 75,
                height: 75,
                entityType: 'EXTERNAL_ENTITIES',
            },
        },
        {
            id: 'e8dabcb7-f471-414e-a999-fe91be5a28fa',
            type: 'group',
            children: [
                'e337f873-64d8-46be-84ed-2a0c38c75fac',
                '09134b5d-8c12-41e8-821b-c97a5a1331c9',
            ],
            group: true,
            label: 'stackrox',
            style: { padding: 15 },
            data: {
                collapsible: true,
                showContextMenu: false,
                entityType: 'NAMESPACE',
            },
        },
        {
            id: 'EXTERNAL',
            type: 'group',
            children: ['__MzQuMTIwLjAuMC8xNg', 'afa12424-bde3-4313-b810-bb463cbe8f90'],
            group: true,
            label: 'External to cluster',
            style: { padding: 15 },
            data: {
                collapsible: true,
                showContextMenu: false,
                entityType: 'EXTERNAL',
            },
        },
    ],
    edges: [
        {
            id: 'e1',
            type: 'edge',
            source: 'e337f873-64d8-46be-84ed-2a0c38c75fac',
            target: '09134b5d-8c12-41e8-821b-c97a5a1331c9',
        },
        {
            id: 'e2',
            type: 'edge',
            source: '09134b5d-8c12-41e8-821b-c97a5a1331c9',
            target: 'e337f873-64d8-46be-84ed-2a0c38c75fac',
        },
        {
            id: 'e3',
            type: 'edge',
            source: '__MzQuMTIwLjAuMC8xNg',
            target: 'e337f873-64d8-46be-84ed-2a0c38c75fac',
        },
        {
            id: 'e4',
            type: 'edge',
            source: 'afa12424-bde3-4313-b810-bb463cbe8f90',
            target: 'e337f873-64d8-46be-84ed-2a0c38c75fac',
        },
    ],
};

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
};

export type TopologyComponentProps = NetworkGraphProps;

const TopologyComponent = ({ detailType, detailId }: TopologyComponentProps) => {
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

const NetworkGraph = React.memo<NetworkGraphProps>(({ detailType, detailId }) => {
    const controller = new Visualization();
    controller.registerLayoutFactory(defaultLayoutFactory);
    controller.registerComponentFactory(defaultComponentFactory);
    controller.registerComponentFactory(stylesComponentFactory);

    return (
        <div className="pf-ri__topology-demo">
            <VisualizationProvider controller={controller}>
                <TopologyComponent detailType={detailType} detailId={detailId} />
            </VisualizationProvider>
        </div>
    );
});

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
