import uniq from 'lodash/uniq';
import flatMap from 'lodash/flatMap';
import findIndex from 'lodash/findIndex';

import featureFlags from 'utils/featureFlags';
import entityTypes from 'constants/entityTypes';
import { filterModes } from 'Containers/Network/Graph/filterModes';

const edgeTypes = {
    NAMESPACE_EDGE: 'NAMESPACE_EDGE',
};

export const isNonIsolatedNode = (node) => node.nonIsolatedIngress && node.nonIsolatedEgress;

export const isDeployment = (node) => node?.type === entityTypes.DEPLOYMENT;

export const isNamespace = (node) => node?.type === entityTypes.NAMESPACE;

export const isNamespaceEdge = (edge) => edge?.type === edgeTypes.NAMESPACE_EDGE;

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object[]} nodes list of nodes
 * @returns {!Object[]}
 */
export const getLinks = (nodes, networkEdgeMap, networkNodeMap) => {
    const filteredLinks = [];

    nodes.forEach((node) => {
        if (node?.entity?.type !== entityTypes.DEPLOYMENT || !networkEdgeMap) {
            return;
        }
        const { id: srcDeploymentId, deployment: srcDeployment } = node.entity;
        const sourceNS = srcDeployment?.namespace;

        const isActive = (key) => !!networkEdgeMap[key]?.active;
        const isNonIsolated = (id) => !!networkNodeMap[id]?.nonIsolated;
        const isBetweenNonIsolated = (srcId, tgtId) => isNonIsolated(srcId) && isNonIsolated(tgtId);
        const isAllowed = (key, { source, target, targetNS }) =>
            sourceNS === 'stackrox' ||
            targetNS === 'stackrox' ||
            isBetweenNonIsolated(source, target) ||
            !!networkEdgeMap[key]?.allowed;
        const isDisallowed = (key, link) =>
            featureFlags.SHOW_DISALLOWED_CONNECTIONS && isActive(key) && !isAllowed(key, link);

        // For nodes that are egress non-isolated, add outgoing edges to ingress non-isolated nodes, as long as the pair
        // of nodes is not fully non-isolated. This is a compromise to make the non-isolation highlight only apply in
        // the case when there are neither ingress nor egress policies (the data sent from the backend is optimized to
        // treat both phenomena separately and omit edges from a egress non-isolated to an ingress non-isolated
        // deployment, but that would be to confusing in the UI).
        if (node.nonIsolatedEgress) {
            nodes.forEach((targetNode) => {
                if (
                    Object.is(node, targetNode) ||
                    targetNode?.entity?.type !== entityTypes.DEPLOYMENT ||
                    !targetNode.nonIsolatedIngress // nodes that are ingress-isolated have explicit incoming edges
                ) {
                    return;
                }

                const { id: tgtDeploymentId, deployment: tgtDeployment } = targetNode.entity;
                const targetNS = tgtDeployment?.namespace;
                const key = [srcDeploymentId, tgtDeploymentId].sort().join('--');

                const link = {
                    source: srcDeploymentId,
                    target: tgtDeploymentId,
                    sourceName: srcDeployment.name,
                    targetName: tgtDeployment.name,
                    sourceNS,
                    targetNS,
                };

                link.isActive = isActive(key);
                link.isBetweenNonIsolated = isBetweenNonIsolated(srcDeploymentId, tgtDeploymentId);
                link.isDisallowed = isDisallowed(key, link);

                // Do not draw implicit links between fully non-isolated nodes unless the connection is active.
                const isImplicit = node.nonIsolatedIngress && targetNode.nonIsolatedEgress;
                if (!isImplicit || link.isActive) {
                    filteredLinks.push(link);
                }
            });
        }

        Object.keys(node.outEdges).forEach((targetIndex) => {
            const tgtNode = nodes[targetIndex];
            if (tgtNode?.entity?.type !== entityTypes.DEPLOYMENT) {
                return;
            }
            const { id: tgtDeploymentId, deployment: tgtDeployment } = tgtNode.entity;
            const targetNS = tgtDeployment?.namespace;
            const key = [srcDeploymentId, tgtDeploymentId].sort().join('--');
            const link = {
                source: srcDeploymentId,
                target: tgtDeploymentId,
                sourceName: node.entity.deployment.name,
                targetName: tgtDeployment.name,
                sourceNS,
                targetNS,
            };

            link.isActive = isActive(key);
            link.isBetweenNonIsolated = isBetweenNonIsolated(srcDeploymentId, tgtDeploymentId);
            link.isDisallowed = isDisallowed(key, link);

            filteredLinks.push(link);
        });
    });

    return filteredLinks;
};

/**
 * Checks against nodeSideMap to return the closest side of the NS that a deployment is positioned
 *
 * @param {!string} source source deployment
 * @param {!string} target target deployment
 * @param {!Object} nodeSideMap map of least distanced sides between source and target deployments
 * @returns {!Object}
 */
export const getSideMap = (source, target, nodeSideMap) => {
    return nodeSideMap?.[source]?.[target] ? nodeSideMap[source][target] : null;
};

/**
 * Iterates through a mapping of classes to boolean types to return a string of appended classes
 *
 * @param {!Object} map object containing className to boolean properties
 * @returns {!string}
 *
 * ex:
 *  input: map = {
 *      isActive: true,
 *      isUnidirectional: false
 *  }
 * output: 'isActive isUnidirectional'
 */
export const getClasses = (map) => {
    return Object.entries(map)
        .filter((entry) => entry[1])
        .map((entry) => entry[0])
        .join(' ');
};

/**
 * Iterates through a list of links and returns bundled edges between namespaces
 *
 * @param {string} highlightedNodeId the hovered or selected node
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap
 * @returns {!Object[]} list of objects describing bundled edges between namespaces
 */
export const getNamespaceEdges = (
    highlightedNodeId,
    { nodes = [], links = [], filterState, nodeSideMap }
) => {
    const delimiter = '**__**';

    const visitedNodeLinks = {};
    const disallowedNamespaceLinks = {};
    const activeNamespaceLinks = {};
    const namespaceLinks = {};
    const linkPortsAndProtocols = {};

    const filteredLinks = links.filter(
        ({ source, target, isActive, sourceNS, targetNS }) =>
            (!highlightedNodeId ||
                source === highlightedNodeId ||
                target === highlightedNodeId ||
                sourceNS === highlightedNodeId ||
                targetNS === highlightedNodeId) &&
            (filterState !== filterModes.active || isActive) &&
            sourceNS &&
            targetNS &&
            sourceNS !== targetNS
    );

    // create a mapping of node edges -> ports and protocols
    nodes.forEach((sourceNode) => {
        const targetNodeIndices = Object.keys(sourceNode.outEdges);
        targetNodeIndices.forEach((targetNodeIndex) => {
            const { properties } = sourceNode.outEdges[targetNodeIndex];
            const targetNode = nodes[targetNodeIndex];
            if (
                sourceNode.entity.type === entityTypes.DEPLOYMENT &&
                targetNode.entity.type === entityTypes.DEPLOYMENT
            ) {
                const nodeLinkKey = [sourceNode.entity.id, targetNode.entity.id]
                    .sort()
                    .join(delimiter);
                linkPortsAndProtocols[nodeLinkKey] = properties;
            }
        });
    });

    filteredLinks.forEach(({ source, target, sourceNS, targetNS, isActive, isDisallowed }) => {
        const namespaceLinkKey = [sourceNS, targetNS].sort().join(delimiter);
        const nodeLinkKey = [source, target].sort().join(delimiter);

        // keep track of which namespace links are active
        if (isActive) activeNamespaceLinks[namespaceLinkKey] = true;
        // keep track of which namespace links are disallowed
        if (isDisallowed) disallowedNamespaceLinks[namespaceLinkKey] = true;

        const portsAndProtocols = linkPortsAndProtocols[nodeLinkKey] || [];
        const isLinkPreviouslyVisited = visitedNodeLinks[nodeLinkKey];

        const namespaceLink = namespaceLinks[namespaceLinkKey] || {
            portsAndProtocols: [],
            numBidirectionalLinks: 0,
            numUnidirectionalLinks: 0,
        };

        namespaceLink.portsAndProtocols = [
            ...namespaceLink.portsAndProtocols,
            ...portsAndProtocols,
        ];

        if (isLinkPreviouslyVisited) {
            namespaceLink.numBidirectionalLinks += 1;
            namespaceLink.numUnidirectionalLinks -= 1;
        } else {
            namespaceLink.numUnidirectionalLinks += 1;
        }

        namespaceLinks[namespaceLinkKey] = namespaceLink;
    });

    return Object.keys(namespaceLinks).map((namespaceLinkKey) => {
        const [sourceNS, targetNS] = namespaceLinkKey.split(delimiter);
        const { portsAndProtocols, numBidirectionalLinks, numUnidirectionalLinks } = namespaceLinks[
            namespaceLinkKey
        ];

        const isNamespaceActive = activeNamespaceLinks[namespaceLinkKey];
        const isNamespaceEdgeActive = filterState !== filterModes.allowed && isNamespaceActive;
        const isNamespaceEdgeDisallowed = disallowedNamespaceLinks[namespaceLinkKey];

        const classes = getClasses({
            namespace: true,
            active: isNamespaceEdgeActive,
            disallowed: isNamespaceEdgeActive && isNamespaceEdgeDisallowed,
        });

        const { source, target } = getSideMap(sourceNS, targetNS, nodeSideMap) || {
            source: sourceNS,
            target: targetNS,
        };

        return {
            data: {
                source,
                target,
                numBidirectionalLinks,
                numUnidirectionalLinks,
                count: numBidirectionalLinks + numUnidirectionalLinks,
                portsAndProtocols,
                type: edgeTypes.NAMESPACE_EDGE,
            },
            classes,
        };
    });
};

/**
 * Iterates through links to return edges that are connected to a node
 *
 * @param {!string} nodeId node id
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap
 * @returns {!Object[]}
 */
export const getEdgesFromNode = (
    nodeId,
    { filterState, links, nodeSideMap, hoveredNode, selectedNode }
) => {
    // to prevent rerendering of duplicate edges
    const edgeMap = {};
    const inAllowedState = filterState === filterModes.allowed;
    links.forEach((linkItem) => {
        const { source, sourceNS, sourceName, target, targetNS, targetName } = linkItem;
        const { isActive, isDisallowed, isBetweenNonIsolated } = linkItem;
        const isSourceNode = nodeId === source;
        const isTargetNode = nodeId === target;
        // if the currently hovered/selected node is a target for this link (ingress)
        const isIngress = hoveredNode?.id === target || selectedNode?.id === target;
        // if the currently hovered/selected node is a source for this link (egress)
        const isEgress = hoveredNode?.id === source || selectedNode?.id === source;
        // destination node info needed for network flow tab
        const destNodeId = isSourceNode ? target : source;
        const destNodeNS = isSourceNode ? targetNS : sourceNS;
        const destNodeName = isSourceNode ? targetName : sourceName;
        if ((isSourceNode || isTargetNode) && (filterState !== filterModes.active || isActive)) {
            const coreClasses = {
                edge: true,
                active: !inAllowedState && isActive,
                // only hide edge when it's bw nonisolated and is not active
                nonIsolated: isBetweenNonIsolated && (!isActive || inAllowedState),
                // an edge is disallowed when it is active but is not allowed
                disallowed: !inAllowedState && isDisallowed,
            };
            const directionalClasses = {
                // if ingress or egress, show edge arrow to indicate direction
                unidirectional: isIngress || isEgress,
            };
            const inSameNS = sourceNS === targetNS;
            const edgeKey = [source, target].sort().join('--');
            if (inSameNS) {
                if (!edgeMap[edgeKey]) {
                    const classes = getClasses({
                        ...coreClasses,
                        ...directionalClasses,
                    });
                    edgeMap[edgeKey] = {
                        data: {
                            destNodeId,
                            destNodeNS,
                            destNodeName,
                            ...linkItem,
                        },
                        classes,
                    };
                } else if (!edgeMap[edgeKey].data.isBidirectional) {
                    // if this edge is already in the edgeMap, it means it's going in the other direction
                    edgeMap[edgeKey].data.isBidirectional = true;
                    edgeMap[edgeKey].classes = getClasses({ ...coreClasses, bidirectional: true });
                }
            } else {
                // make sure both nodes have edges drawn to the nearest side of their NS
                let sourceNSSide = sourceNS;
                let targetNSSide = targetNS;
                const sideMap = getSideMap(sourceNS, targetNS, nodeSideMap);
                if (sideMap) {
                    sourceNSSide = sideMap.source;
                    targetNSSide = sideMap.target;
                }

                const isWithinSourceNS =
                    hoveredNode?.parent === sourceNS ||
                    (!hoveredNode && selectedNode?.parent === sourceNS);
                const isWithinTargetNS =
                    hoveredNode?.parent === targetNS ||
                    (!hoveredNode && selectedNode?.parent === targetNS);

                const innerSourceEdgeKey = [source, sourceNSSide].sort().join('--');
                const innerTargetEdgeKey = [targetNSSide, target].sort().join('--');

                if (!edgeMap[innerSourceEdgeKey]) {
                    // if the inner edge from source/target to namespace is in the same namespace as selected
                    const classes = getClasses({
                        ...coreClasses,
                        ...directionalClasses,
                        inner: true,
                        withinNS: isWithinSourceNS,
                    });
                    // Edge from source deployment to it's namespace edge
                    edgeMap[innerSourceEdgeKey] = {
                        data: {
                            source,
                            target: sourceNSSide,
                            isDisallowed,
                        },
                        classes,
                    };
                } else if (!edgeMap[innerSourceEdgeKey].data.isBidirectional && !isWithinSourceNS) {
                    // if this edge is already in the edgeMap, it means it's going in the other direction
                    edgeMap[innerSourceEdgeKey].data.isBidirectional = true;
                    edgeMap[innerSourceEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                    });
                }

                if (!edgeMap[innerTargetEdgeKey]) {
                    const classes = getClasses({
                        ...coreClasses,
                        ...directionalClasses,
                        inner: true,
                        withinNS: isWithinTargetNS,
                    });

                    // Edge from namespace edge to target deployment
                    edgeMap[innerTargetEdgeKey] = {
                        data: {
                            source: targetNSSide,
                            target,
                            destNodeId,
                            destNodeName,
                            destNodeNS,
                            isActive,
                            isDisallowed,
                        },
                        classes,
                    };
                } else if (!edgeMap[innerTargetEdgeKey].data.isBidirectional && !isWithinTargetNS) {
                    // if this edge is already in the edgeMap, it means it's going in the other direction
                    edgeMap[innerTargetEdgeKey].data.isBidirectional = true;
                    edgeMap[innerTargetEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                    });
                }
            }
        }
    });
    return Object.values(edgeMap);
};

/**
 * Iterates through a list of nodes to return a list of deployments with proper styling classes for cytoscape
 *
 * @param {!Object[]} filteredData list of deployments
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap,
 *                            networkNodeMap, hoveredNode, and selectedNode
 * @returns {!Object[]}
 */
export const getDeploymentList = (filteredData, configObj) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;
    const deploymentList = filteredData.map((datum) => {
        const { entity, ...datumProps } = datum;
        const { deployment, ...entityProps } = entity;
        const { namespace, ...deploymentProps } = deployment;

        const edges = getEdgesFromNode(entityProps.id, configObj);

        const isSelected = !!(selectedNode?.id === entity.id);
        const isHovered = !!(hoveredNode?.id === entity.id);
        const isBackground =
            !(selectedNode === undefined && hoveredNode === undefined) && !isHovered && !isSelected;
        const isNonIsolated = isNonIsolatedNode(datum);
        const isDisallowed =
            filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
        const classes = getClasses({
            active: datum.isActive,
            selected: isSelected,
            deployment: true,
            disallowed: isDisallowed,
            hovered: isHovered,
            background: isBackground,
            nonIsolated: isNonIsolated,
        });

        let ingress = [];
        let egress = [];
        const entityData = networkNodeMap[entity.id];
        if (entityData) {
            const {
                ingressAllowed = [],
                ingressActive = [],
                egressAllowed = [],
                egressActive = [],
            } = entityData;
            if (filterState === filterModes.allowed) {
                ingress = ingressAllowed;
                egress = egressAllowed;
            } else if (filterState === filterModes.active) {
                ingress = ingressActive;
                egress = egressActive;
            } else {
                ingress = [...ingressActive, ...ingressAllowed];
                egress = [...egressActive, ...egressAllowed];
            }
        }

        const deploymentNode = {
            data: {
                ...datumProps,
                ...entityProps,
                ...deploymentProps,
                parent: namespace,
                edges,
                deploymentId: entityProps.id,
                ingress,
                egress,
            },
            classes,
        };
        return deploymentNode;
    });
    return deploymentList;
};

/**
 * Iterates through the list of nodes to return the data of a single deployment
 *
 * @param {!string} id node id
 * @param {!Object[]} deploymentList list of deployments
 * @returns {!Object[]}
 */
export const getNodeData = (id, deploymentList) => {
    return deploymentList.filter((node) => node.data.deploymentId === id);
};

/**
 * Iterates through a list of links and returns all links for the currently interacted node
 *
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap,
 *                            hoveredNode, and selectedNode
 * @returns {!Object[]}
 */
export const getEdges = (configObj) => {
    const { hoveredNode, selectedNode } = configObj;
    const highlightedNode = hoveredNode || selectedNode;
    let allEdges = getNamespaceEdges(highlightedNode?.id, configObj);
    if (highlightedNode) {
        allEdges = [...allEdges, ...getEdgesFromNode(highlightedNode.id, configObj)];
    }
    return allEdges;
};

/**
 * Iterates through the nodes to return a list of namespaces with active deployments
 *
 * @param {!Object} filteredData nodes that pertain to deployments
 * @param {!Object[]} deploymentList list of deployments
 * @returns {!Object[]}
 */
export const getActiveNamespaceList = (filteredData, deploymentList) => {
    return uniq(
        filteredData.reduce((acc, curr) => {
            const nsName = curr.entity.deployment.namespace;
            if (
                deploymentList.some(
                    (element) => element.data.isActive && element.data.parent === nsName
                )
            ) {
                acc.push(nsName);
            }

            return acc;
        }, [])
    );
};

/**
 * Iterates through a list of nodes to return a list of namespaces enriched by styling classes
 *
 * @param {!Object} filteredData nodes that pertain to deployments
 * @param {!Object[]} deploymentList list of deployments
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains hoveredNode, and selectedNode
 * @returns {!Object[]}
 */
export const getNamespaceList = (filteredData, deploymentList, { hoveredNode, selectedNode }) => {
    const activeNamespaceList = getActiveNamespaceList(filteredData, deploymentList);
    return uniq(filteredData.map((datum) => datum.entity.deployment.namespace)).map((namespace) => {
        const isActive = activeNamespaceList.includes(namespace);
        const isHovered = hoveredNode?.id === namespace || hoveredNode?.parent === namespace;
        const isSelected = selectedNode?.id === namespace || selectedNode?.parent === namespace;
        const isBackground =
            !(selectedNode === undefined && hoveredNode === undefined) && !isHovered && !isSelected;
        const classes = getClasses({
            nsActive: isActive,
            nsSelected: isSelected,
            nsHovered: isHovered,
            background: isBackground,
        });

        return {
            data: {
                id: namespace,
                name: `${isActive ? '\ue901' : ''} ${namespace}`,
                active: isActive,
                type: entityTypes.NAMESPACE,
            },
            classes,
        };
    });
};

/**
 * Returns a list of nodes that are hidden namespace cardinal direction edges
 *
 * @param {!Object[]} namespaceList list of namespaces
 * @returns {!Object[]}
 */
const sides = ['top', 'left', 'right', 'bottom'];

export const getNamespaceEdgeNodes = (namespaceList) => {
    const nodes = [];
    namespaceList.forEach((namespace) => {
        const nsName = namespace.data.id;
        sides.forEach((side) => {
            nodes.push({
                data: {
                    id: `${nsName}_${side}`,
                    parent: nsName,
                    side,
                },
                classes: 'nsEdge',
            });
        });
    });
    return nodes;
};

/**
 * Iterates through a list of active nodes and returns nodes with active network policies
 *
 * @param {!Object[]} activeNodes list of active nodes
 * @param {!Object[]} allowedNodes list of allowed nodes
 * @returns {!Object[]}
 */
const getActiveNetPolNodes = (activeNodes, allowedNodes) => {
    return activeNodes.map((activeNode) => {
        const node = { ...activeNode };
        const matchedNode = allowedNodes.find(
            // default true for allowedNode.entity.id to prevent nullish equality
            (allowedNode) => (allowedNode?.entity?.id ?? true) === node?.entity?.id
        );
        if (!matchedNode) {
            return node;
        }
        node.policyIds = flatMap(matchedNode.policyIds);
        return node;
    });
};

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object[]} activeNodes list of active nodes
 * @param {!Object[]} allowedNodes list of allowed nodes
 * @param {string} filterState current filter state of the network graph
 * @returns {!Object[]}
 */
export const getFilteredNodes = (activeNodes, allowedNodes, filterState) => {
    if (filterState !== filterModes.active) {
        return allowedNodes;
    }

    // return as is
    if (!allowedNodes || !activeNodes) {
        return activeNodes;
    }

    return getActiveNetPolNodes(activeNodes, allowedNodes);
};

/**
 * Iterates through a list of nodes and returns only ports and protocols for have edges that go
 * to a specific node
 *
 * @param {!Object[]} nodes list of nodes
 * @param {Object} node the target node
 * @returns {!Object[]}
 */
export const getIngressPortsAndProtocols = (nodes, node = {}) => {
    const { id, ingress } = node;

    if (!id || !ingress || ingress.length === 0) return [];

    const targetNodeIndex = findIndex(nodes, (n) => {
        return n.id === id;
    });

    const ingressNodes = nodes.filter((n) => ingress.indexOf(n.id) !== -1);

    const ingressPortsAndProtocols = ingressNodes.reduce((acc, curr) => {
        if (curr?.outEdges[targetNodeIndex]) {
            return [...acc, ...curr.outEdges[targetNodeIndex].properties];
        }
        return acc;
    }, []);

    return ingressPortsAndProtocols;
};

/**
 * Iterates through a list of nodes and returns only ports and protocols for edges that go
 * out of a specific node
 *
 * @param {!Object[]} outEdges node's outedges
 * @returns {!Object[]}
 */
export const getEgressPortsAndProtocols = (outEdges) => {
    if (!outEdges || outEdges.length === 0) return [];
    return Object.values(outEdges).reduce((acc, curr) => {
        return [...acc, ...curr.properties];
    }, []);
};
