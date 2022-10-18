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

import './Topology.css';

import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';
import DeploymentSideBar from './deployment/DeploymentSideBar';
import NamespaceSideBar from './namespace/NamespaceSideBar';

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

const TopologyComponent = () => {
    const [selectedObj, setSelectedObj] = useState<NodeModel | EdgeModel | null>(null);

    const controller = useVisualizationController();

    React.useEffect(() => {
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
                    {selectedObj?.type === 'group' && <NamespaceSideBar />}
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
