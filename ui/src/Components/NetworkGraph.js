import React, { useState, useRef, useEffect } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import Cytoscape from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import { uniq, throttle, flatMap } from 'lodash';
import popper from 'cytoscape-popper';
/* Cannot use neither Tooltip nor HoverHint components as Cytoscape renders on
canvas (no DOM elements). Instead using 'cytoscape-popper' and  special
configuration of 'tippy.js' instance to position the tooltip. */
// eslint-disable-next-line no-restricted-imports
import tippy from 'tippy.js';

import { actions as graphActions } from 'reducers/network/graph';
import GraphLoader from 'Containers/Network/Graph/Overlays/GraphLoader';
import { edgeGridLayout, getParentPositions } from 'Containers/Network/Graph/networkGraphLayouts';
import { filterModes } from 'Containers/Network/Graph/filterModes';
import style from 'Containers/Network/Graph/networkGraphStyles';
import { getLinks, nonIsolated, isNamespace } from 'utils/networkGraphUtils';
import { NS_FONT_SIZE, MAX_ZOOM, MIN_ZOOM, ZOOM_STEP, GRAPH_PADDING } from 'constants/networkGraph';
import entityTypes from 'constants/entityTypes';
import { defaultTippyTooltipProps } from 'Components/Tooltip';

function getClasses(map) {
    return Object.entries(map)
        .filter(entry => entry[1])
        .map(entry => entry[0])
        .join(' ');
}

Cytoscape.use(popper);
Cytoscape('layout', 'edgeGridLayout', edgeGridLayout);
Cytoscape.use(edgeGridLayout);

function getNodesForFilterState(activeNodes, allowedNodes, filterState) {
    if (filterState !== filterModes.active) {
        return allowedNodes;
    }

    // return as is
    if (!allowedNodes || !activeNodes) {
        return activeNodes;
    }

    const activeNodesWithNetPol = activeNodes.map(activeNode => {
        const node = { ...activeNode };
        const matchedNode = allowedNodes
            .map(n => n)
            .find(
                allowedNode =>
                    allowedNode.entity && node.entity && allowedNode.entity.id === node.entity.id
            );
        if (!matchedNode) {
            return node;
        }
        node.policyIds = flatMap(matchedNode.policyIds);
        return node;
    });

    return activeNodesWithNetPol;
}

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
    match
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

    const nodes = getNodesForFilterState(activeNodes, allowedNodes, filterState);
    const data = nodes.map(datum => ({
        ...datum,
        isActive: filterState !== filterModes.active && datum.internetAccess
    }));

    const links = getLinks(data, networkEdgeMap, networkNodeMap);

    function makePopperDiv(text) {
        const div = document.createElement('div');
        // theoretically we can use `TooltipOverlay` component with ReactDOM.createPortal,
        // yet not clear how to hook React lifecycle into Cytoscape one
        div.classList.add('rox-tooltip-overlay');
        div.innerHTML = text;
        document.body.appendChild(div);
        return div;
    }

    function getSideMap(source, target) {
        return nodeSideMap && nodeSideMap[source] && nodeSideMap[source][target]
            ? nodeSideMap[source][target]
            : null;
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
            }
        });

        tippyRef.current.show();
    }

    function getNSEdges(nodeId) {
        const delimiter = '**__**';

        const filteredLinks = links.filter(
            ({ source, target, isActive, sourceNS, targetNS }) =>
                (!nodeId ||
                    source === nodeId ||
                    target === nodeId ||
                    sourceNS === nodeId ||
                    targetNS === nodeId) &&
                (filterState !== filterModes.active || isActive) &&
                sourceNS &&
                targetNS &&
                sourceNS !== targetNS
        );

        const sourceTargetMap = {};
        const disallowedLinkMap = {};
        const activeLinkMap = filteredLinks.reduce((acc, curr) => {
            const { sourceNS, targetNS, isActive, isDisallowed } = curr;
            const key = [sourceNS, targetNS].sort().join(delimiter);
            if (isActive) acc[key] = true;
            if (isDisallowed) {
                disallowedLinkMap[key] = true;
            }
            return acc;
        }, {});

        const counts = filteredLinks.reduce((acc, curr) => {
            const sourceTargetKey = [curr.source, curr.target].sort().join(delimiter);
            if (sourceTargetMap[sourceTargetKey]) {
                return acc;
            }

            sourceTargetMap[sourceTargetKey] = true;
            const key = [curr.sourceNS, curr.targetNS].sort().join(delimiter);
            acc[key] = acc[key] ? acc[key] + 1 : 1;
            return acc;
        }, {});

        return Object.keys(counts).map(key => {
            const [sourceId, targetId] = key.split(delimiter);
            const count = counts[key];
            const isActive = activeLinkMap[key];
            const activeClass = filterState !== filterModes.allowed && isActive ? 'active' : '';
            const disallowedClass =
                filterState !== filterModes.allowed && (isActive && disallowedLinkMap[key])
                    ? 'disallowed'
                    : '';
            const { source, target } = getSideMap(sourceId, targetId) || {
                sourceId,
                targetId
            };

            return {
                data: {
                    source,
                    target,
                    count
                },
                classes: `namespace ${activeClass} ${disallowedClass}`
            };
        });
    }

    function getEdgesFromNode(nodeId) {
        const edgeMap = {};
        const edges = [];
        const inAllowedState = filterState === filterModes.allowed;
        links.forEach(linkItem => {
            const { source, sourceNS, sourceName, target, targetNS, targetName } = linkItem;
            const { isActive, isDisallowed, isBetweenNonIsolated } = linkItem;
            const nodeIsSource = nodeId === source;
            const nodeIsTarget = nodeId === target;
            // destination node info needed for network flow tab
            const destNodeId = nodeIsSource ? target : source;
            const destNodeNS = nodeIsSource ? targetNS : sourceNS;
            const destNodeName = nodeIsSource ? targetName : sourceName;
            if (
                (nodeIsSource || nodeIsTarget) &&
                (filterState !== filterModes.active || isActive)
            ) {
                const activeClass = !inAllowedState && isActive ? 'active' : '';

                // only hide edge when it's bw nonisolated and is not active
                const nonIsolatedClass =
                    isBetweenNonIsolated && (!isActive || inAllowedState) ? 'nonIsolated' : '';
                // an edge is disallowed when it is active but is not allowed
                const disallowedClass = !inAllowedState && isDisallowed ? 'disallowed' : '';
                // if(isDisallowed) console.log(linkItem)
                const classes = `${activeClass} ${nonIsolatedClass} ${disallowedClass}`;
                const id = [source, target].sort().join('--');
                if (!edgeMap[id]) {
                    // If same namespace, draw line between the two nodes
                    if (sourceNS === targetNS) {
                        edges.push({
                            data: {
                                destNodeId,
                                destNodeNS,
                                destNodeName,
                                ...linkItem
                            },
                            classes: `edge ${classes}`
                        });
                    } else {
                        // make sure both nodes have edges drawn to the nearest side of their NS
                        let sourceNSSide = sourceNS;
                        let targetNSSide = targetNS;
                        const sideMap = getSideMap(sourceNS, targetNS);
                        if (sideMap) {
                            sourceNSSide = sideMap.source;
                            targetNSSide = sideMap.target;
                        }

                        // Edge from source to it's namespace
                        edges.push({
                            data: {
                                source,
                                target: sourceNSSide,
                                isDisallowed
                            },
                            classes: `edge inner ${classes}`
                        });

                        // Edge from target to its namespace
                        edges.push({
                            data: {
                                source: target,
                                target: targetNSSide,
                                destNodeId,
                                destNodeName,
                                destNodeNS,
                                isActive,
                                isDisallowed
                            },
                            classes: `edge inner ${classes}`
                        });
                    }
                    edgeMap[id] = true;
                }
            }
        });
        return edges;
    }

    function getDeploymentsList() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = filteredData.map(datum => {
            const { entity, ...datumProps } = datum;
            const { deployment, ...entityProps } = entity;
            const { namespace, ...deploymentProps } = deployment;

            const edges = getEdgesFromNode(entityProps.id, true);

            const isSelected = !!(selectedNode && selectedNode.id === entity.id);
            const isHovered = !!(hoveredNode && hoveredNode.id === entity.id);
            const isBackground =
                !(selectedNode === undefined && hoveredNode === undefined) &&
                !isHovered &&
                !isSelected;
            const isNonIsolated = nonIsolated(datum);
            const isDisallowed =
                filterState !== filterModes.allowed && edges.find(edge => edge.data.isDisallowed);
            const classes = getClasses({
                active: datum.isActive,
                selected: isSelected,
                deployment: true,
                disallowed: isDisallowed,
                hovered: isHovered,
                background: isBackground,
                nonIsolated: isNonIsolated
            });

            const deploymentNode = {
                data: {
                    ...datumProps,
                    ...entityProps,
                    ...deploymentProps,
                    parent: namespace,
                    edges,
                    deploymentId: entityProps.id
                },
                classes
            };
            return deploymentNode;
        });
        return deploymentList;
    }

    function getNodes() {
        const filteredData = data.filter(datum => datum.entity && datum.entity.deployment);
        const deploymentList = getDeploymentsList();
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
        ).map(namespace => {
            const active = activeNamespaces.includes(namespace);
            const isHovered =
                hoveredNode && (hoveredNode.id === namespace || hoveredNode.parent === namespace);
            const isSelected =
                selectedNode &&
                (selectedNode.id === namespace || selectedNode.parent === namespace);
            const isBackground =
                !(selectedNode === undefined && hoveredNode === undefined) &&
                !isHovered &&
                !isSelected;
            const classes = getClasses({
                nsActive: active,
                nsSelected: isSelected,
                nsHovered: isHovered,
                background: isBackground
            });

            return {
                data: {
                    id: namespace,
                    name: `${active ? '\ue901' : ''} ${namespace}`,
                    active,
                    type: entityTypes.NAMESPACE
                },
                classes
            };
        });

        const namespaceEdgeNodes = namespaceList.reduce((acc, namespace) => {
            const nsName = namespace.data.id;
            const set = ['top', 'left', 'right', 'bottom'];

            const newNodes = set.map(side => ({
                data: {
                    id: `${nsName}_${side}`,
                    parent: nsName,
                    side
                },
                classes: 'nsEdge'
            }));
            return acc.concat(newNodes);
        }, []);

        namespaceList.forEach(namespace => {
            deploymentList.forEach(deployment => {
                if (!namespacesWithDeployments[namespace.data.id]) {
                    namespacesWithDeployments[namespace.data.id] = [];
                }
                if (deployment.data.parent === namespace.data.id) {
                    namespacesWithDeployments[namespace.data.id].push(deployment);
                }
            });
        });

        return [...namespaceList, ...deploymentList, ...namespaceEdgeNodes];
    }

    function getEdges() {
        const node = hoveredNode || selectedNode;
        let allEdges = getNSEdges(node && node.id);
        if (node) {
            allEdges = allEdges.concat(getEdgesFromNode(node.id));
        }
        return allEdges;
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

    function getNodeData(id) {
        return getDeploymentsList().filter(node => node.data.deploymentId === id);
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
            renderedPosition: { x: cyRef.current.width() / 2, y: cyRef.current.height() / 2 }
        });
    }

    function zoomOut() {
        if (!cyRef.current) return;

        cyRef.current.zoom({
            level: Math.min(cyRef.current.zoom() - ZOOM_STEP, MAX_ZOOM),
            renderedPosition: { x: cyRef.current.width() / 2, y: cyRef.current.height() / 2 }
        });
    }

    function getElements() {
        return { nodes: getNodes(), edges: getEdges() };
    }

    // Calculate which namespace box side combinations are shortest and store them
    function calculateNodeSideMap(changedNodeId) {
        if (!cyRef.current) return;

        // Get a map of all the side nodes per namespace
        const namespaces = cyRef.current.nodes(':parent');
        const sideNodesPerParent = namespaces.reduce((acc, namespace) => {
            const { id } = namespace.data(); // to

            const sideNodes = cyRef.current.nodes(`[parent="${id}"][side]`);

            const nodesInfo = sideNodes.map(node => {
                const { x, y } = node.position();
                const { side } = node.data();
                return {
                    node,
                    side,
                    id: node.id(),
                    x,
                    y
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
                sourceSideNodes.forEach(sourceSideNode => {
                    const sourceSide = sourceSideNode.side;
                    const targetSideNode = targetSideNodes.find(tgtNode => {
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
                            distance
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
        const newEdges = getEdges();

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
        window.addEventListener('resize', throttle(() => zoomToFit, 100));

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
            onNodeClick
        });
    }

    function runLayout() {
        if (!cyRef.current) return;
        const CY = cyRef.current;
        const NSPositions = getParentPositions(CY.nodes(), { x: 100, y: 100 }); // all nodes, padding

        NSPositions.forEach(position => {
            const { id, x, y } = position;
            CY.layout({
                name: 'edgeGridLayout',
                parentPadding: { bottom: 5, top: 0, left: 0, right: 0 },
                position: { x, y },
                eles: CY.nodes(`[parent="${id}"]`)
            }).run();
        });
        CY.fit(null, GRAPH_PADDING);
        const node = getNodeData(match.params.deploymentId);
        if (setSelectedNodeInGraph && node.length) {
            setSelectedNodeInGraph(node[0].data);
            setSelectedNode(node[0].data);
            onNodeClick(node[0].data);
        }
        if (selectedNode && isNamespace(selectedNode)) {
            onNamespaceClick({
                id: selectedNode.id,
                deployments: namespacesWithDeployments[selectedNode.id] || []
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
                    name: PropTypes.string.isRequired
                })
            }).isRequired
        })
    ).isRequired,
    allowedNodes: PropTypes.arrayOf(
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
    simulatorOn: PropTypes.bool.isRequired
};

NetworkGraph.defaultProps = {
    setSelectedNodeInGraph: null,
    networkEdgeMap: {}
};

const mapDispatchToProps = {
    setNetworkGraphRef: graphActions.setNetworkGraphRef,
    setSelectedNamespace: graphActions.setSelectedNamespace,
    setSelectedNodeInGraph: graphActions.setSelectedNode
};

export default withRouter(
    connect(
        null,
        mapDispatchToProps
    )(NetworkGraph)
);
