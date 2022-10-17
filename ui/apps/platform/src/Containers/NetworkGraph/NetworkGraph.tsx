import React, { useState } from 'react';
import { useHistory } from 'react-router-dom';
import {
    EdgeModel,
    Model,
    NodeModel,
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

function findEntityById(graphModel, id) {
    let entity = null;
    entity = graphModel.nodes?.find((node) => node.id === id);
    if (!entity) {
        entity = graphModel.groups?.find((group) => group.id === id);
    }
    return entity;
}

function getUrlParamsForEntity(selectEntity): [string, string] {
    const urlDetailTypes = {
        NAMESPACE: 'namespace',
        DEPLOYMENT: 'deployment',
        CIDR_BLOCK: 'cidr',
        EXTERNAL_ENTITIES: 'internet',
        EXTERNAL: 'external',
    };
    const detailType = urlDetailTypes[selectEntity.data.entityType];
    const detailId = selectEntity.id;

    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
    return [detailType, detailId];
}

const TopologyComponent = () => {
    const [selectedObj, setSelectedObj] = useState<NodeModel | EdgeModel | null>(null);
    const history = useHistory();

    const controller = useVisualizationController();

    React.useEffect(() => {
        function onSelect(ids: string[]) {
            const newSelectedId = ids?.[0] || null;
            // check if selected id is for a node
            // const newSelectedEntity = model.nodes?.find((node) => node.id === newSelectedId);
            const newSelectedEntity = newSelectedId && findEntityById(model, newSelectedId);
            const [detailType, detailId] = getUrlParamsForEntity(newSelectedEntity);
            if (newSelectedEntity && (newSelectedEntity as any)?.data?.entityType !== 'EXTERNAL') {
                setSelectedObj(newSelectedEntity);
                history.push(`${networkBasePathPF}/${detailType}/${detailId}`);
            } else {
                setSelectedObj(null);
                history.push(`${networkBasePathPF}`);
            }
        }

        controller.fromModel(model, false);
        controller.addEventListener(SELECTION_EVENT, onSelect);

        return () => {
            controller.removeEventListener(SELECTION_EVENT, onSelect);
        };
    }, [controller, history]);

    const selectedIds = selectedObj ? [selectedObj.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={() => setSelectedObj(null)}>
                    {selectedObj?.data?.entityType === 'NAMESPACE' && <NamespaceSideBar />}
                    {selectedObj?.data?.entityType === 'DEPLOYMENT' && <DeploymentSideBar />}
                    {selectedObj?.data?.entityType === 'CIDR_BLOCK' && <CidrBlockSideBar />}
                    {selectedObj?.data?.entityType === 'EXTERNAL_ENTITIES' && (
                        <ExternalEntitiesSideBar />
                    )}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedObj}
            sideBarResizable
        >
            <VisualizationSurface state={{ selectedIds }} />
        </TopologyView>
    );
};

const NetworkGraph: React.FunctionComponent = React.memo(() => {
    const controller = new Visualization();
    controller.registerLayoutFactory(defaultLayoutFactory);
    controller.registerComponentFactory(defaultComponentFactory);
    controller.registerComponentFactory(stylesComponentFactory);

    return (
        <div className="pf-ri__topology-demo">
            <VisualizationProvider controller={controller}>
                <TopologyComponent />
            </VisualizationProvider>
        </div>
    );
});

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
