import React, { useState } from 'react';
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

import './Topology.css';

import stylesComponentFactory from './components/stylesComponentFactory';
import defaultLayoutFactory from './layouts/defaultLayoutFactory';
import defaultComponentFactory from './components/defaultComponentFactory';

const TopologyComponent = () => {
    const [selectedId, setSelectedId] = useState<string | null>(null);

    const controller = useVisualizationController();

    React.useEffect(() => {
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
                        collapsible: false,
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

        function onSelect(ids: string[]) {
            const newSelectedId = ids?.[0] || null;
            setSelectedId(newSelectedId);
        }

        controller.fromModel(model, false);
        controller.addEventListener(SELECTION_EVENT, onSelect);

        return () => {
            controller.removeEventListener(SELECTION_EVENT, onSelect);
        };
    }, [controller]);

    return (
        <TopologyView
            sideBar={
                <TopologySideBar show={!!selectedId} resizable onClose={() => setSelectedId(null)}>
                    <div style={{ height: '100%' }}>{selectedId}</div>
                </TopologySideBar>
            }
            sideBarOpen={!!selectedId}
            sideBarResizable
        >
            <VisualizationSurface />
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
