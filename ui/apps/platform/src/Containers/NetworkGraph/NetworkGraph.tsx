import React, { useState } from 'react';
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

import { SearchFilter } from 'types/search';

import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import DeploymentSideBar from './deployment/DeploymentSideBar';

import './Topology.css';

const model: Model = {
    graph: {
        id: 'g1',
        type: 'graph',
        layout: 'ColaNoForce',
    },
    nodes: [
        {
            id: 'n1',
            label: 'Central',
            type: 'node',
            width: 75,
            height: 75,
            data: {
                id: 'n1',
                label: 'Central',
                type: 'node',
                width: 75,
                height: 75,
            },
        },
        {
            id: 'n2',
            label: 'Sensor',
            type: 'node',
            width: 75,
            height: 75,
            data: {
                id: 'n2',
                label: 'Sensor',
                type: 'node',
                width: 75,
                height: 75,
            },
        },
        {
            id: 'group1',
            type: 'group',
            children: ['n1', 'n2'],
            group: true,
            label: 'stackrox',
            style: { padding: 15 },
            data: {
                collapsible: true,
                showContextMenu: false,
            },
        },
    ],
    edges: [
        {
            id: 'e1',
            type: 'edge',
            source: 'n1',
            target: 'n2',
        },
        {
            id: 'e2',
            type: 'edge',
            source: 'n2',
            target: 'n1',
        },
    ],
};

const TopologyComponent = ({ searchFilter }: NetworkGraphProps) => {
    const [selectedObj, setSelectedObj] = useState<NodeModel | EdgeModel | null>(null);
    const [selectedId, setSelectedId] = useState<string | null>(null);

    const controller = useVisualizationController();

    React.useEffect(() => {
        const model: Model = {
            graph: {
                id: 'g1',
                type: 'graph',
                layout: 'ColaGroupsLayout',
            },
            nodes: [
                {
                    id: 'n1',
                    label: 'Central',
                    type: 'node',
                    width: 75,
                    height: 75,
                    data: {
                        id: 'n1',
                        label: 'Central',
                        type: 'node',
                        width: 75,
                        height: 75,
                        nodeType: 'DEPLOYMENT',
                    },
                },
                {
                    id: 'n2',
                    label: 'Sensor',
                    type: 'node',
                    width: 75,
                    height: 75,
                    data: {
                        id: 'n2',
                        label: 'Sensor',
                        type: 'node',
                        width: 75,
                        height: 75,
                        nodeType: 'DEPLOYMENT',
                    },
                },
                {
                    id: 'group1',
                    type: 'group',
                    children: ['n1', 'n2'],
                    group: true,
                    label: 'stackrox',
                    style: { padding: 15 },
                    data: {
                        collapsible: false,
                        showContextMenu: false,
                        nodeType: 'NAMESPACE',
                    },
                },
                {
                    id: 'n3',
                    label: 'Google/us-central1 | 34.72.0.0/16',
                    type: 'node',
                    width: 75,
                    height: 75,
                    data: {
                        id: 'n3',
                        label: 'Google/us-central1 | 34.72.0.0/16',
                        type: 'node',
                        width: 150,
                        height: 150,
                        nodeType: 'DEPLOYMENT',
                    },
                },
            ],
            edges: [
                {
                    id: 'e1',
                    type: 'edge',
                    source: 'n1',
                    target: 'n2',
                },
                {
                    id: 'e2',
                    type: 'edge',
                    source: 'n2',
                    target: 'n1',
                },
                {
                    id: 'e3',
                    type: 'edge',
                    source: 'n1',
                    target: 'n3',
                },
            ],
        };

        function onSelect(ids: string[]) {
            const newSelectedId = ids?.[0] || null;
            // check if selected id is for a node
            const newSelectedNode = model.nodes?.find((node) => node.id === newSelectedId);
            if (newSelectedNode) {
                setSelectedObj(newSelectedNode);
            }
            // if not then do nothing
        }

        controller.fromModel(model, false);
        controller.addEventListener(SELECTION_EVENT, onSelect);

        return () => {
            controller.removeEventListener(SELECTION_EVENT, onSelect);
        };
    }, [controller]);

    const selectedIds = selectedObj ? [selectedObj.id] : [];

    return (
        <TopologyView
            sideBar={
                <TopologySideBar resizable onClose={() => setSelectedObj(null)}>
                    {selectedObj?.type === 'group' && <div>Group</div>}
                    {selectedObj?.type === 'node' && <DeploymentSideBar />}
                </TopologySideBar>
            }
            sideBarOpen={!!selectedObj}
            sideBarResizable
        >
            <VisualizationSurface state={{ selectedIds }} />
        </TopologyView>
    );
};

type NetworkGraphProps = {
    searchFilter?: SearchFilter;
    detailType?: string;
    detailId?: string;
};

const NetworkGraph: React.FunctionComponent<NetworkGraphProps> = React.memo(
    ({ searchFilter, detailType, detailId }: NetworkGraphProps) => {
        const controller = new Visualization();
        controller.registerLayoutFactory(defaultLayoutFactory);
        controller.registerComponentFactory(defaultComponentFactory);
        controller.registerComponentFactory(stylesComponentFactory);

        return (
            <div className="pf-ri__topology-demo">
                <VisualizationProvider controller={controller}>
                    <TopologyComponent
                        searchFilter={searchFilter}
                        detailType={detailType}
                        detailId={detailId}
                    />
                </VisualizationProvider>
            </div>
        );
    }
);

NetworkGraph.displayName = 'NetworkGraph';

export default NetworkGraph;
