import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Cytoscape from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import { throttle } from 'lodash';
import popper from 'cytoscape-popper';
/* Cannot use neither Tooltip nor HoverHint components as Cytoscape renders on
canvas (no DOM elements). Instead using 'cytoscape-popper' and  special
configuration of 'tippy.js' instance to position the tooltip. */
// eslint-disable-next-line no-restricted-imports
import tippy from 'tippy.js';
import ReactDOM from 'react-dom';

import { actions as graphActions } from 'reducers/network/graph';
import GraphLoader from 'Containers/Network/Graph/Overlays/GraphLoader';
import { edgeGridLayout, getParentPositions } from 'Containers/Network/Graph/networkGraphLayouts';
import { filterModes } from 'Containers/Network/Graph/filterModes';
import style from 'Containers/Network/Graph/networkGraphStyles';
import {
    getLinks,
    isNamespace,
    getNodeData,
    getEdges,
    getNamespaceEdgeNodes,
    getNamespaceList,
    getDeploymentList,
    getFilteredNodes,
} from 'utils/networkGraphUtils';
import { NS_FONT_SIZE, MAX_ZOOM, MIN_ZOOM, ZOOM_STEP, GRAPH_PADDING } from 'constants/networkGraph';
import { defaultTippyTooltipProps } from 'Components/Tooltip';
import TooltipOverlay from 'Components/TooltipOverlay';

Cytoscape.use(popper);
Cytoscape('layout', 'edgeGridLayout', edgeGridLayout);
Cytoscape.use(edgeGridLayout);

const NetworkGraph = ({
    activeNodes,
    allowedNodes,
    networkEdgeMap,
    networkNodeMap,
    onNodeClick,
    onNamespaceClick,
    onClickOutside,
    filterState,
    setNetworkGraphRef,
    setSelectedNamespace,
    setSelectedNodeInGraph,
    simulatorOn,
    history,
    match,
}) => {
    const [selectedNode, setSelectedNode] = useState();
    const [hoveredNode, setHoveredNode] = useState();
    const [firstRenderFinished, setFirstRenderFinished] = useState(false);
    const nodeSideMapRef = useRef({});
    const zoomFontMapRef = useRef({});
    const nodeSideMap = nodeSideMapRef.current;
    const zoomFontMap = zoomFontMapRef.current;
    const cyRef = useRef();
    const tippyRef = useRef();
    const namespacesWithDeployments = {};

    const nodes = getFilteredNodes(activeNodes, allowedNodes, filterState);
    const data = nodes.map((datum) => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess,
    }));

    const links = getLinks(data, networkEdgeMap, networkNodeMap);

    function makePopperDiv(text) {
        const div = document.createElement('div');
        ReactDOM.render(<TooltipOverlay>{text}</TooltipOverlay>, div);
        return div;
    }

    function createTippy(elm, text) {
        if (!elm) return;
        const popperRef = elm.popperRef();
        if (tippyRef.current) tippyRef.current.destroy();

        tippyRef.current = tippy(document.createElement('div'), {
            content: makePopperDiv(text),
            ...defaultTippyTooltipProps,
            lazy: false,
            onCreate(instance) {
                // eslint-disable-next-line no-param-reassign
                instance.popperInstance.reference = popperRef;
            },
        });

        tippyRef.current.show();
    }

    function nodeHoverHandler(ev) {
        const node = ev.target.data();
        const { id, name, parent, side } = node;
        const isChild = !!parent;
        if (!cyRef || !isChild || side) return;

        setHoveredNode(node);
        const nodeElm = cyRef.current.getElementById(id);
        const parentElm = cyRef.current.getElementById(parent);
        createTippy(nodeElm, name);
        const children = parentElm.descendants();
        children.removeClass('background');
    }

    function nodeMouseOutHandler() {
        setHoveredNode();
    }

    function clickHandler(ev) {
        const { target } = ev;
        const evData = target.data && target.data();
        const id = evData && evData.id;
        const isParent = target.isParent && target.isParent();
        const isEdge = target.isEdge && target.isEdge();

        // Canvas or Selected node click: clear selection
        if (!id || !evData || (selectedNode && evData && id === selectedNode.id)) {
            setSelectedNode();
            onClickOutside();
            history.push('/main/network');
            return;
        }

        // Edge click or edge node click
        if (isEdge || evData.side) return;

        // Parent Click
        if (isParent) {
            if (id) {
                onNamespaceClick({ id, deployments: namespacesWithDeployments[id] || [] });
                setSelectedNode(evData);
            }
            return;
        }

        // Node click: select node
        if (target.isNode()) {
            setSelectedNode(evData);
            history.push(`/main/network/${evData.id}`);
            onNodeClick(evData);
        }

        if (!isParent) {
            setSelectedNamespace(null);
        }
    }

    function zoomHandler() {
        if (!cyRef || !cyRef.current) return;

        // to dynamically set the font size of namespace labels
        const zoomConstant = 20;
        const curZoomLevel = Math.round(cyRef.current.zoom() * zoomConstant);
        if (!zoomFontMap[curZoomLevel]) {
            zoomFontMap[curZoomLevel] = Math.max(
                (NS_FONT_SIZE / curZoomLevel) * zoomConstant,
                NS_FONT_SIZE
            );
        }
        cyRef.current.nodes(':parent').style('font-size', zoomFontMap[curZoomLevel]);
        cyRef.current.edges('.namespace').style('font-size', zoomFontMap[curZoomLevel]);
    }

    function zoomToFit() {
        if (!cyRef) return;
        cyRef.current.fit(null, GRAPH_PADDING);
        const newMinZoom = Math.min(cyRef.current.zoom(), cyRef.current.minZoom());
        cyRef.current.minZoom(newMinZoom);
        zoomHandler();
    }

    function zoomIn() {
        if (!cyRef.current) return;

        cyRef.current.zoom({
            level: Math.max(cyRef.current.zoom() + ZOOM_STEP, cyRef.current.minZoom()),
            renderedPosition: { x: cyRef.current.width() / 2, y: cyRef.current.height() / 2 },
        });
    }

    function zoomOut() {
        if (!cyRef.current) return;

        cyRef.current.zoom({
            level: Math.min(cyRef.current.zoom() - ZOOM_STEP, MAX_ZOOM),
            renderedPosition: { x: cyRef.current.width() / 2, y: cyRef.current.height() / 2 },
        });
    }

    function getConfigObj() {
        return {
            hoveredNode,
            selectedNode,
            links,
            filterState,
            nodeSideMap,
            networkNodeMap,
        };
    }

    function getElements() {
        const configObj = getConfigObj();
        const filteredData = data.filter((datum) => datum.entity && datum.entity.deployment);
        const deploymentList = getDeploymentList(filteredData, configObj);
        const namespaceList = getNamespaceList(filteredData, deploymentList, configObj);
        const namespaceEdgeNodes = getNamespaceEdgeNodes(namespaceList);

        namespaceList.forEach((namespace) => {
            deploymentList.forEach((deployment) => {
                if (!namespacesWithDeployments[namespace.data.id]) {
                    namespacesWithDeployments[namespace.data.id] = [];
                }
                if (deployment.data.parent === namespace.data.id) {
                    namespacesWithDeployments[namespace.data.id].push(deployment);
                }
            });
        });
        return {
            nodes: [...namespaceList, ...deploymentList, ...namespaceEdgeNodes],
            edges: getEdges(configObj),
        };
    }

    // Calculate which namespace box side combinations are shortest and store them
    function calculateNodeSideMap(changedNodeId) {
        if (!cyRef.current) return;

        // Get a map of all the side nodes per namespace
        const namespaces = cyRef.current.nodes(':parent');
        const sideNodesPerParent = namespaces.reduce((acc, namespace) => {
            const { id } = namespace.data(); // to

            const sideNodes = cyRef.current.nodes(`[parent="${id}"][side]`);

            const nodesInfo = sideNodes.map((node) => {
                const { x, y } = node.position();
                const { side } = node.data();
                return {
                    node,
                    side,
                    id: node.id(),
                    x,
                    y,
                };
            });
            return { ...acc, [id]: nodesInfo };
        }, {});

        const distances = {};

        function getDistance(sourceSideNode, targetSideNode) {
            const key = [sourceSideNode.id, targetSideNode.id].sort().join('**__**');
            const cachedDistance = distances[key];
            if (cachedDistance) return cachedDistance;
            const dX = Math.abs(sourceSideNode.x - targetSideNode.x);
            const dY = Math.abs(sourceSideNode.y - targetSideNode.y);
            const distance = Math.sqrt(dX * dX + dY * dY);
            distances[key] = distance;
            return distance;
        }
        // for each namespace, go through each other namespace
        namespaces.forEach((sourceNS, i) => {
            const sourceName = sourceNS.data().id;
            const sourceSideNodes = sideNodesPerParent[sourceName];
            nodeSideMap[sourceName] = nodeSideMap[sourceName] || {};
            const sourceMap = nodeSideMap[sourceName];

            namespaces.forEach((targetNS, j) => {
                const targetName = targetNS.data().id;

                if (i === j || (changedNodeId && ![sourceName, targetName].includes(changedNodeId)))
                    return;

                const targetSideNodes = sideNodesPerParent[targetName];
                let shortest;
                // check distances between every combination of side nodes to find shortest
                sourceSideNodes.forEach((sourceSideNode) => {
                    const sourceSide = sourceSideNode.side;
                    const targetSideNode = targetSideNodes.find((tgtNode) => {
                        const { side } = tgtNode;
                        if (sourceSide === 'top') return side === 'bottom';
                        if (sourceSide === 'bottom') return side === 'top';
                        if (sourceSide === 'left') return side === 'right';
                        if (sourceSide === 'right') return side === 'left';
                        return false;
                    });

                    const distance = getDistance(sourceSideNode, targetSideNode);
                    if (!shortest || shortest.distance > distance) {
                        shortest = {
                            source: sourceSideNode.id,
                            target: targetSideNode.id,
                            sourceSide: sourceSideNode.side,
                            targetSide: targetSideNode.side,
                            distance,
                        };
                    }
                });
                sourceMap[targetName] = shortest;
            });
        });
    }

    function handleDrag(ev) {
        let changedNodeId;
        if (ev && ev.target) changedNodeId = ev.target.data().id;

        calculateNodeSideMap(changedNodeId);
        const configObj = getConfigObj();
        const newEdges = getEdges(configObj);

        cyRef.current.remove('edge');
        cyRef.current.add(newEdges);
    }

    function configureCY(cyInstance) {
        cyRef.current = cyInstance;
        cyRef.current
            .off('click mouseover mouseout mousedown drag')
            .on('click', clickHandler)
            .on('mouseover', 'node', throttle(nodeHoverHandler, 100))
            .on('mouseout', 'node', nodeMouseOutHandler)
            .on('mouseout mousedown', 'node', () => {
                if (tippyRef.current) tippyRef.current.destroy();
            })
            .on('drag', throttle(handleDrag, 100))
            .on('zoom', zoomHandler)
            .ready(() => {
                if (firstRenderFinished) return;
                zoomToFit();
                setFirstRenderFinished(true);
            });
    }

    const elements = getElements();
    // Effects
    function setWindowResize() {
        window.addEventListener(
            'resize',
            throttle(() => zoomToFit, 100)
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
            setSelectedNode,
            selectedNode,
            getNodeData,
            onNodeClick,
        });
    }

    function runLayout() {
        if (!cyRef.current) return;
        const CY = cyRef.current;
        const NSPositions = getParentPositions(CY.nodes(), { x: 100, y: 100 }); // all nodes, padding

        NSPositions.forEach((position) => {
            const { id, x, y } = position;
            CY.layout({
                name: 'edgeGridLayout',
                parentPadding: { bottom: 5, top: 0, left: 0, right: 0 },
                position: { x, y },
                eles: CY.nodes(`[parent="${id}"]`),
            }).run();
        });
        CY.fit(null, GRAPH_PADDING);
        const configObj = getConfigObj();
        const filteredData = data.filter((datum) => datum.entity && datum.entity.deployment);
        const deploymentList = getDeploymentList(filteredData, configObj);
        const node = getNodeData(match.params.deploymentId, deploymentList);
        if (setSelectedNodeInGraph && node.length) {
            setSelectedNodeInGraph(node[0].data);
            setSelectedNode(node[0].data);
            onNodeClick(node[0].data);
        }
        if (selectedNode && isNamespace(selectedNode)) {
            onNamespaceClick({
                id: selectedNode.id,
                deployments: namespacesWithDeployments[selectedNode.id] || [],
            });
        }
        if (simulatorOn) {
            setSelectedNode();
            setSelectedNamespace(null);
        }
    }

    function grabifyNamespaces() {
        if (!cyRef.current) return;
        const CY = cyRef.current;
        CY.nodes(`[parent]`).ungrabify();
    }

    useEffect(setWindowResize, []);
    useEffect(setGraphRef, []);
    useEffect(runLayout, [activeNodes, allowedNodes, networkEdgeMap, filterState, simulatorOn]);
    useEffect(grabifyNamespaces);
    useEffect(calculateNodeSideMap);

    const normalizedElements = CytoscapeComponent.normalizeElements(elements);

    const loader = !normalizedElements && (
        <div className="absolute flex h-full items-center justify-center top-0 w-full pointer-events-none">
            <GraphLoader isLoading />
        </div>
    );

    return (
        <div className="h-full w-full relative">
            <div
                id="cytoscapeContainer"
                className="w-full h-full cursor-pointer cytoscape-container"
            >
                <CytoscapeComponent
                    elements={normalizedElements}
                    layout={{ name: 'grid' }}
                    stylesheet={style}
                    cy={configureCY}
                    minZoom={MIN_ZOOM}
                    maxZoom={MAX_ZOOM}
                    style={{ width: '100%', height: '100%' }}
                />
            </div>
            {loader}
        </div>
    );
};

NetworkGraph.propTypes = {
    activeNodes: PropTypes.arrayOf(
        PropTypes.shape({
            entity: PropTypes.shape({
                type: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired,
                deployment: PropTypes.shape({
                    name: PropTypes.string.isRequired,
                }),
            }).isRequired,
        })
    ).isRequired,
    allowedNodes: PropTypes.arrayOf(
        PropTypes.shape({
            entity: PropTypes.shape({
                type: PropTypes.string.isRequired,
                id: PropTypes.string.isRequired,
                deployment: PropTypes.shape({
                    name: PropTypes.string.isRequired,
                }),
            }).isRequired,
        })
    ).isRequired,
    networkEdgeMap: PropTypes.shape({}),
    networkNodeMap: PropTypes.shape({}).isRequired,
    onNamespaceClick: PropTypes.func.isRequired,
    onNodeClick: PropTypes.func.isRequired,
    onClickOutside: PropTypes.func.isRequired,
    filterState: PropTypes.number.isRequired,
    setNetworkGraphRef: PropTypes.func.isRequired,
    setSelectedNamespace: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    setSelectedNodeInGraph: PropTypes.func,
    simulatorOn: PropTypes.bool.isRequired,
};

NetworkGraph.defaultProps = {
    setSelectedNodeInGraph: null,
    networkEdgeMap: {},
};

const mapDispatchToProps = {
    setNetworkGraphRef: graphActions.setNetworkGraphRef,
    setSelectedNamespace: graphActions.setSelectedNamespace,
    setSelectedNodeInGraph: graphActions.setSelectedNode,
};

export default withRouter(connect(null, mapDispatchToProps)(NetworkGraph));
