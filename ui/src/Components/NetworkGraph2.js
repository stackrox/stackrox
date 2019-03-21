import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as graphActions } from 'reducers/network/graph';

import Cytoscape from 'cytoscape';
import coseBilkentPlugin from 'cytoscape-cose-bilkent';
import { coseBilkent as layout } from 'Containers/Network/Graph/networkGraphLayouts';

import { getLinks } from 'utils/networkGraphUtils';
import { uniq, debounce } from 'lodash';

import filterModes from 'Containers/Network/Graph/filterModes';
import style from 'Containers/Network/Graph/networkGraphStyles';
import CytoscapeComponent from 'react-cytoscapejs';

import { MAX_ZOOM, MIN_ZOOM, ZOOM_STEP, GRAPH_PADDING } from 'constants/cytoscapeGraph';

Cytoscape.use(coseBilkentPlugin);

function getClasses(map) {
    return Object.entries(map)
        .filter(entry => entry[1])
        .map(entry => entry[0])
        .join(' ');
}

const NetworkGraph = ({
    nodes,
    networkFlowMapping,
    onNodeClick,
    filterState,
    setNetworkGraphRef
}) => {
    const [selectedNode, setSelectedNode] = useState();
    const [hoveredNode, setHoveredNode] = useState();
    const cy = useRef();

    const data = nodes.map(datum => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess
    }));

    function getEdgesFromNode(nodeId) {
        const edges = getLinks(data, networkFlowMapping)
            .filter(linkItem => !nodeId || linkItem.source === nodeId || linkItem.target === nodeId) // filter by specific nodeId
            .filter(linkItem => filterState !== filterModes.active || linkItem.isActive)
            .map(linkItem => ({
                data: linkItem,
                classes: linkItem.isActive ? 'active' : ''
            }));

        return edges;
    }

    function getNodes() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = filteredData.map(datum => {
            const { entity, ...datumProps } = datum;
            const { deployment, ...entityProps } = entity;
            const { namespace: parent, ...deploymentProps } = deployment;
            const isSelected = selectedNode && selectedNode.id === entity.id;
            const classes = getClasses({
                active: datum.isActive,
                selected: isSelected
            });

            return {
                data: {
                    ...datumProps,
                    ...entityProps,
                    ...deploymentProps,
                    parent,
                    deploymentId: entityProps.id
                },
                classes
            };
        });

        const activeNamespaces = filteredData.reduce((acc, curr) => {
            const nsName = curr.entity.deployment.namespace;
            if (
                deploymentList.find(
                    element => element.data.isActive && element.data.parent === nsName
                )
            ) {
                acc.push(nsName);
            }

            return acc;
        }, []);

        const namespaceList = uniq(
            filteredData.map(datum => datum.entity.deployment.namespace)
        ).map(namespace => ({
            data: {
                id: namespace
            },
            classes: activeNamespaces.includes(namespace) ? 'nsActive' : ''
        }));

        return [...namespaceList, ...deploymentList];
    }

    function getEdges() {
        if (hoveredNode || selectedNode) {
            const node = selectedNode || hoveredNode;
            return getEdgesFromNode(node.id);
        }

        return [];
    }

    function nodeHoverHandler(ev) {
        setHoveredNode(ev.target.data());
    }

    function nodeMouseOutHandler() {
        setHoveredNode();
    }

    function clickHandler(ev) {
        // Canvas or Selected node click: clear selection
        if (
            !ev.target.data ||
            (selectedNode && ev.target.data() && ev.target.data().id === selectedNode.id)
        ) {
            setSelectedNode();
            return;
        }

        // Parent Click: Do nothing
        if (ev.target.isParent()) {
            return;
        }

        // Node click: select node
        const node = ev.target.data();
        setSelectedNode(node);
        onNodeClick(node);
    }

    function zoomToFit() {
        if (!cy) return;

        cy.current.fit(null, GRAPH_PADDING);
    }

    function zoomIn() {
        if (!cy.current) return;

        cy.current.zoom({
            level: Math.max(cy.current.zoom() + ZOOM_STEP, MIN_ZOOM),
            position: { x: 0, y: 0 }
        });
        cy.current.center();
    }

    function zoomOut() {
        if (!cy.current) return;

        cy.current.zoom({
            level: Math.min(cy.current.zoom() - ZOOM_STEP, MAX_ZOOM),
            position: { x: 0, y: 0 }
        });
        cy.current.center();
    }

    function getElements() {
        return { nodes: getNodes(), edges: getEdges() };
    }

    function configureCY(cyInstance) {
        cy.current = cyInstance;
        cyInstance
            .on('click', null, ev => {
                clickHandler(ev);
            })
            .on('mouseover', 'node', debounce(nodeHoverHandler, 100))
            .on('mouseout', 'node', nodeMouseOutHandler);
    }

    const elements = getElements();

    // Effects
    function handleWindowResize() {
        window.addEventListener(
            'resize',
            debounce(() => {
                if (cy.current) cy.current.fit(null, GRAPH_PADDING);
            }, 100)
        );

        const cleanup = () => {
            window.removeEventListener('resize');
        };

        return cleanup;
    }

    function setGraphRef() {
        setNetworkGraphRef({
            zoomToFit,
            zoomIn,
            zoomOut,
            setSelectedNode
        });
    }

    function runLayout() {
        if (!cy.current) return;
        cy.current.layout(layout).run();
    }

    useEffect(handleWindowResize, []);
    useEffect(setGraphRef, []);
    useEffect(runLayout, [nodes.length]);

    return (
        <div className="h-full w-full relative">
            <div id="cytoscapeContainer" className="w-full h-full">
                <CytoscapeComponent
                    elements={CytoscapeComponent.normalizeElements(elements)}
                    layout={layout}
                    stylesheet={style}
                    cy={configureCY}
                    minZoom={MIN_ZOOM}
                    maxZoom={MAX_ZOOM}
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
        </div>
    );
};

NetworkGraph.propTypes = {
    nodes: PropTypes.arrayOf(
        PropTypes.shape({
            entity: PropTypes.shape({
                type: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired,
                deployment: PropTypes.shape({
                    name: PropTypes.string.isRequired
                })
            }).isRequired
        })
    ).isRequired,
    networkFlowMapping: PropTypes.shape({}).isRequired,
    onNodeClick: PropTypes.func.isRequired,
    filterState: PropTypes.number.isRequired,
    setNetworkGraphRef: PropTypes.func.isRequired
};

const mapDispatchToProps = {
    setNetworkGraphRef: graphActions.setNetworkGraphRef
};

export default connect(
    null,
    mapDispatchToProps
)(NetworkGraph);
