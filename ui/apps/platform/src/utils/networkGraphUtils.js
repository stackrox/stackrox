import uniq from 'lodash/uniq';
import flatMap from 'lodash/flatMap';

import { UIfeatureFlags, isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';
import entityTypes from 'constants/entityTypes';
import { networkTraffic, networkConnections, EXTERNAL_ENTITIES_TYPE } from 'constants/networkGraph';
import { filterModes } from 'constants/networkFilterModes';

export const edgeTypes = {
    NAMESPACE_EDGE: 'NAMESPACE_EDGE',
    NODE_TO_NODE_EDGE: 'NODE_TO_NODE_EDGE',
    NODE_TO_NAMESPACE_EDGE: 'NODE_TO_NAMESPACE_EDGE',
};
const LINK_DELIMITER = '**__**';

export const isNonIsolatedNode = (node) => node.nonIsolatedIngress && node.nonIsolatedEgress;

export const isDeployment = (node) => node?.type === entityTypes.DEPLOYMENT;

export const isNamespace = (node) => node?.type === entityTypes.NAMESPACE;

export const isNamespaceEdge = (edge) => edge?.type === edgeTypes.NAMESPACE_EDGE;

export const isNodeToNodeEdge = (edge) => edge?.type === edgeTypes.NODE_TO_NODE_EDGE;

export const isNodeToNamespaceEdge = (edge) => edge?.type === edgeTypes.NODE_TO_NAMESPACE_EDGE;

/**
 * Create a key using a source and target with a delimiter in between
 *
 * @param {!string} source a string representing the source node
 * @param {!string} target a string representing the target node
 * @returns {!string}
 *
 * ex: getSourceTargetKey("source", "target") => "source**__**target"
 */
export const getSourceTargetKey = (source, target) => {
    return [source, target].sort().join(LINK_DELIMITER);
};

/**
 * Gets the source and target from a node link key
 *
 * @param {!string} sourceTargetKey a string representing a key using a source and target
 * @returns {!String[]}
 *
 * ex: getSourceTargetFromKey("source**__**target") => ["source", "target"]
 */
export const getSourceTargetFromKey = (sourceTargetKey) => {
    return sourceTargetKey.split(LINK_DELIMITER);
};

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object[]} nodes list of nodes
 * @param {!Object} networkEdgeMap map of edges in the graph by srcId--tgtId key
 * @param {!Object} networkNodeMap map of nodes in the graph by nodeId
 * @param {!string} filterState current filter state
 * @returns {!Object[]}
 */
export const getLinks = (nodes, networkEdgeMap, networkNodeMap, filterState) => {
    const filteredLinks = [];
    // a map of all the edges in the node set to know whether we need to add disallowed edges
    const filteredEdgeHashTable = {};

    const isActive = (edgeKey) => !!networkEdgeMap[edgeKey]?.active;
    const isNonIsolated = (nodeId) => !!networkNodeMap[nodeId]?.nonIsolated;
    const isBetweenNonIsolated = (sourceId, targetId) =>
        isNonIsolated(sourceId) && isNonIsolated(targetId);
    const isAllowed = (edgeKey, { source, target, targetNS, sourceNS }) =>
        sourceNS === 'stackrox' ||
        targetNS === 'stackrox' ||
        isBetweenNonIsolated(source, target) ||
        !!networkEdgeMap[edgeKey]?.allowed;
    const isDisallowed = (edgeKey, link) =>
        UIfeatureFlags.SHOW_DISALLOWED_CONNECTIONS &&
        isActive(edgeKey) &&
        !isAllowed(edgeKey, link);

    nodes.forEach((node) => {
        if (node?.entity?.type !== entityTypes.DEPLOYMENT || !networkEdgeMap) {
            return;
        }

        const { id: sourceDeploymentId, deployment: sourceDeployment } = node.entity;
        const sourceNS = sourceDeployment?.namespace;

        // For nodes that are egress non-isolated, add outgoing edges to ingress non-isolated nodes, as long as the pair
        // of nodes is not fully non-isolated. This is a compromise to make the non-isolation highlight only apply in
        // the case when there are neither ingress nor egress policies (the data sent from the backend is optimized to
        // treat both phenomena separately and omit edges from a egress non-isolated to an ingress non-isolated
        // deployment, but that would be too confusing in the UI).
        if (node.nonIsolatedEgress) {
            nodes.forEach((targetNode) => {
                if (
                    Object.is(node, targetNode) ||
                    targetNode?.entity?.type !== entityTypes.DEPLOYMENT ||
                    !targetNode.nonIsolatedIngress // nodes that are ingress-isolated have explicit incoming edges
                ) {
                    return;
                }

                const { id: targetDeploymentId, deployment: targetDeployment } = targetNode.entity;
                const targetNS = targetDeployment?.namespace;
                const edgeKey = getSourceTargetKey(sourceDeploymentId, targetDeploymentId);

                const link = {
                    source: sourceDeploymentId,
                    target: targetDeploymentId,
                    sourceName: sourceDeployment.name,
                    targetName: targetDeployment.name,
                    sourceNS,
                    targetNS,
                };

                link.isActive = isActive(edgeKey);
                link.isBetweenNonIsolated = isBetweenNonIsolated(
                    sourceDeploymentId,
                    targetDeploymentId
                );
                link.isAllowed = isAllowed(edgeKey, link);
                link.isDisallowed = isDisallowed(edgeKey, link);

                // Do not draw implicit links between fully non-isolated nodes unless the connection is active.
                const isImplicit = node.nonIsolatedIngress && targetNode.nonIsolatedEgress;
                if (!isImplicit || link.isActive) {
                    filteredLinks.push(link);
                    filteredEdgeHashTable[edgeKey] = true;
                }
            });
        }

        Object.keys(node.outEdges).forEach((targetDeploymentId) => {
            const targetNode = networkNodeMap[targetDeploymentId].active;
            if (targetNode?.entity?.type !== entityTypes.DEPLOYMENT) {
                return;
            }
            const { deployment: targetDeployment } = targetNode.entity;
            const targetNS = targetDeployment?.namespace;
            const edgeKey = getSourceTargetKey(sourceDeploymentId, targetDeploymentId);
            const link = {
                source: sourceDeploymentId,
                target: targetDeploymentId,
                sourceName: node.entity.deployment.name,
                targetName: targetDeployment.name,
                sourceNS,
                targetNS,
            };

            link.isActive = isActive(edgeKey);
            link.isBetweenNonIsolated = isBetweenNonIsolated(
                sourceDeploymentId,
                targetDeploymentId
            );
            link.isAllowed = isAllowed(edgeKey, link);
            link.isDisallowed = isDisallowed(edgeKey, link);

            filteredLinks.push(link);
            filteredEdgeHashTable[edgeKey] = true;
        });

        // if in the All filter state, merge active outEdges for each node from networkNodeMap so
        // we include disallowed edges that are not part of the allowed node set
        if (filterState === filterModes.all) {
            Object.keys(networkNodeMap[sourceDeploymentId].active.outEdges).forEach(
                (targetDeploymentId) => {
                    const targetNode = networkNodeMap[targetDeploymentId];
                    if (targetNode.active.entity.type !== entityTypes.DEPLOYMENT) {
                        return;
                    }
                    const {
                        name: targetName,
                        namespace: targetNS,
                    } = targetNode.active.entity.deployment;
                    const edgeKey = getSourceTargetKey(sourceDeploymentId, targetDeploymentId);
                    // to prevent double counting if an edge is also allowed and active
                    if (!filteredEdgeHashTable[edgeKey]) {
                        const link = {
                            source: sourceDeploymentId,
                            target: targetDeploymentId,
                            sourceName: node.entity.deployment.name,
                            targetName,
                            sourceNS,
                            targetNS,
                        };

                        link.isActive = isActive(edgeKey);
                        link.isBetweenNonIsolated = isBetweenNonIsolated(
                            sourceDeploymentId,
                            targetDeploymentId
                        );
                        link.isAllowed = isAllowed(edgeKey, link);
                        link.isDisallowed = isDisallowed(edgeKey, link);

                        filteredLinks.push(link);
                        filteredEdgeHashTable[edgeKey] = true;
                    }
                }
            );
        }
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
 * Creates a mapping of ports/protocols based on node links (source->target), and then
 * returns a closure to allow getting the ports/protocols of a specific source->target
 *
 * @param {!Object[]} node
 * @param {!String} highlightedNodeId
 * @param {!Object} networkNodeMap
 * @param {!String} filterState
 * @returns {!Object}
 *
 */
export const createPortsAndProtocolsSelector = (
    nodes,
    highlightedNodeId,
    networkNodeMap,
    filterState
) => {
    const linkPortsAndProtocols = {};

    // create a mapping of node edges -> ports and protocols
    nodes.forEach((sourceNode) => {
        const targetNodeIds = Object.keys(sourceNode?.outEdges || {});
        targetNodeIds.forEach((targetNodeId) => {
            if (!networkNodeMap?.[targetNodeId]) {
                return;
            }
            const { allowed, active } = networkNodeMap[targetNodeId];
            let targetNode = allowed;
            if (filterState === filterModes.active) {
                targetNode = active;
            }
            const { properties } = sourceNode.outEdges[targetNodeId];
            if (
                sourceNode.entity.type === entityTypes.DEPLOYMENT &&
                targetNode.entity.type === entityTypes.DEPLOYMENT
            ) {
                const nodeLinkKey = getSourceTargetKey(sourceNode.entity.id, targetNode.entity.id);
                const traffic =
                    targetNode.entity.id === highlightedNodeId
                        ? networkTraffic.INGRESS
                        : networkTraffic.EGRESS;
                const modifiedProperties = properties.map((datum) => {
                    return { ...datum, traffic };
                });
                if (linkPortsAndProtocols[nodeLinkKey]) {
                    linkPortsAndProtocols[nodeLinkKey] = [
                        ...linkPortsAndProtocols[nodeLinkKey],
                        ...modifiedProperties,
                    ];
                } else {
                    linkPortsAndProtocols[nodeLinkKey] = [...modifiedProperties];
                }
            }
        });
    });

    function getPortsAndProtocolsByLink(nodeLinkKey, isEgress) {
        if (linkPortsAndProtocols[nodeLinkKey]) {
            return linkPortsAndProtocols[nodeLinkKey];
        }
        if (typeof isEgress !== 'boolean') {
            throw Error('The value for isEgress must be set');
        }
        // if the mapping doesn't contain the ports/protocols information, it's because we create
        // additional edges between egress non-isolated and ingress non-isolated nodes. For those cases,
        // we want to default to showing Any protocols/ Any ports
        const traffic = isEgress ? 'egress' : 'ingress';
        return [{ port: '*', protocol: 'L4_PROTOCOL_ANY', traffic }];
    }

    return getPortsAndProtocolsByLink;
};

/**
 * Iterates through a list of links and returns bundled edges between namespaces
 *
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, nodeSideMap
 * @returns {!Object[]} list of objects describing bundled edges between namespaces
 */
export const getNamespaceEdges = ({
    nodes = [],
    links = [],
    filterState,
    nodeSideMap,
    selectedNode,
    hoveredNode,
    hoveredEdge,
    networkNodeMap,
}) => {
    const visitedNodeLinks = {};
    const disallowedNamespaceLinks = {};
    const activeNamespaceLinks = {};
    const namespaceLinks = {};
    const highlightedNodeId = (hoveredNode || selectedNode)?.id;
    const getPortsAndProtocolsByLink = createPortsAndProtocolsSelector(
        nodes,
        highlightedNodeId,
        networkNodeMap,
        filterState
    );

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

    filteredLinks.forEach(
        ({ source, target, sourceNS, targetNS, isActive, isAllowed, isDisallowed }) => {
            const namespaceLinkKey = getSourceTargetKey(sourceNS, targetNS);
            const nodeLinkKey = getSourceTargetKey(source, target);
            const isEgress = source === highlightedNodeId;

            // keep track of which namespace links are active
            if (isActive) {
                activeNamespaceLinks[namespaceLinkKey] = true;
            }
            // keep track of which namespace links are disallowed
            if (isDisallowed) {
                disallowedNamespaceLinks[namespaceLinkKey] = true;
            }

            const portsAndProtocols = getPortsAndProtocolsByLink(nodeLinkKey, isEgress);
            const isLinkPreviouslyVisited = visitedNodeLinks[nodeLinkKey];

            const namespaceLink = namespaceLinks[namespaceLinkKey] || {
                portsAndProtocols: [],
                numBidirectionalLinks: 0,
                numUnidirectionalLinks: 0,
                numActiveBidirectionalLinks: 0,
                numActiveUnidirectionalLinks: 0,
                numAllowedBidirectionalLinks: 0,
                numAllowedUnidirectionalLinks: 0,
            };

            namespaceLink.portsAndProtocols = [
                ...namespaceLink.portsAndProtocols,
                ...portsAndProtocols,
            ];

            if (isLinkPreviouslyVisited) {
                namespaceLink.numBidirectionalLinks += 1;
                namespaceLink.numUnidirectionalLinks = namespaceLink.numUnidirectionalLinks
                    ? namespaceLink.numUnidirectionalLinks - 1
                    : 0;
                if (isActive) {
                    namespaceLink.numActiveBidirectionalLinks += 1;
                    namespaceLink.numActiveUnidirectionalLinks = namespaceLink.numActiveUnidirectionalLinks
                        ? namespaceLink.numActiveUnidirectionalLinks - 1
                        : 0;
                }
                if (isAllowed) {
                    namespaceLink.numAllowedBidirectionalLinks += 1;
                    namespaceLink.numAllowedUnidirectionalLinks = namespaceLink.numAllowedUnidirectionalLinks
                        ? namespaceLink.numAllowedUnidirectionalLinks - 1
                        : 0;
                }
            } else {
                namespaceLink.numUnidirectionalLinks += 1;
                if (isActive) {
                    namespaceLink.numActiveUnidirectionalLinks += 1;
                }
                if (isAllowed) {
                    namespaceLink.numAllowedUnidirectionalLinks += 1;
                }
                visitedNodeLinks[nodeLinkKey] = true;
            }

            namespaceLinks[namespaceLinkKey] = namespaceLink;
        }
    );

    return Object.keys(namespaceLinks).map((namespaceLinkKey) => {
        const [sourceNS, targetNS] = getSourceTargetFromKey(namespaceLinkKey);
        const {
            portsAndProtocols,
            numBidirectionalLinks,
            numUnidirectionalLinks,
            numActiveBidirectionalLinks,
            numActiveUnidirectionalLinks,
            numAllowedBidirectionalLinks,
            numAllowedUnidirectionalLinks,
        } = namespaceLinks[namespaceLinkKey];
        const isHoveredEdge =
            (hoveredEdge?.sourceNodeNamespace === sourceNS &&
                hoveredEdge?.targetNodeNamespace === targetNS) ||
            (hoveredEdge?.targetNodeNamespace === sourceNS &&
                hoveredEdge?.sourceNodeNamespace === targetNS);

        const isNamespaceActive = activeNamespaceLinks[namespaceLinkKey];
        const isNamespaceEdgeActive = filterState !== filterModes.allowed && isNamespaceActive;
        const isNamespaceEdgeDisallowed = disallowedNamespaceLinks[namespaceLinkKey];

        const classes = getClasses({
            namespace: true,
            active: isNamespaceEdgeActive,
            disallowed: isNamespaceEdgeActive && isNamespaceEdgeDisallowed,
            hovered: isHoveredEdge,
        });

        const { source, target } = getSideMap(sourceNS, targetNS, nodeSideMap) || {
            source: sourceNS,
            target: targetNS,
        };

        return {
            data: {
                source,
                target,
                sourceNodeNamespace: sourceNS,
                targetNodeNamespace: targetNS,
                numBidirectionalLinks,
                numUnidirectionalLinks,
                numActiveBidirectionalLinks,
                numActiveUnidirectionalLinks,
                numAllowedBidirectionalLinks,
                numAllowedUnidirectionalLinks,
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
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap
 * @returns {!Object[]}
 */
export const getEdgesFromNode = ({
    filterState,
    links,
    nodes,
    nodeSideMap,
    hoveredNode,
    selectedNode,
    hoveredEdge,
    networkNodeMap,
}) => {
    // to prevent rerendering of duplicate edges
    const nodeLinks = {};
    const inAllowedFilterState = filterState === filterModes.allowed;
    const highlightedNode = hoveredNode || selectedNode;

    // if a node wasn't selected or hovered over, we don't want to show it's links
    if (!highlightedNode) {
        return [];
    }

    const getPortsAndProtocolsByLink = createPortsAndProtocolsSelector(
        nodes,
        highlightedNode?.id,
        networkNodeMap,
        filterState
    );

    links.forEach((link) => {
        const {
            source,
            sourceNS,
            sourceName,
            target,
            targetNS,
            targetName,
            isActive,
            isAllowed,
            isDisallowed,
            isBetweenNonIsolated,
        } = link;
        const isSourceNode = highlightedNode?.id === source;
        const isEgress = isSourceNode;
        const isTargetNode = highlightedNode?.id === target;
        // if the currently hovered/selected node is a target for this link (ingress)
        const isRelativeIngress = highlightedNode?.id === target;
        // if the currently hovered/selected node is a source for this link (egress)
        const isRelativeEgress = highlightedNode?.id === source;
        // destination node info needed for network flow tab
        const destNodeId = isSourceNode ? target : source;
        const destNodeName = isSourceNode ? targetName : sourceName;
        const destNodeNamespace = isSourceNode ? targetNS : sourceNS;
        const sourceNodeId = source;
        const sourceNodeName = sourceName;
        const sourceNodeNamespace = sourceNS;
        const targetNodeId = target;
        const targetNodeName = targetName;
        const targetNodeNamespace = targetNS;

        if ((isSourceNode || isTargetNode) && (filterState !== filterModes.active || isActive)) {
            const coreClasses = {
                edge: true,
                active: !inAllowedFilterState && isActive,
                // only hide edge when it's bw nonisolated and is not active
                nonIsolated: isBetweenNonIsolated && (!isActive || inAllowedFilterState),
                // an edge is disallowed when it is active but is not allowed
                disallowed: !inAllowedFilterState && isDisallowed,
            };
            const directionalClasses = {
                // if ingress or egress, show edge arrow to indicate direction
                unidirectional: isRelativeIngress || isRelativeEgress,
            };
            const inSameNamespace = sourceNS === targetNS;
            const nodeLinkKey = getSourceTargetKey(source, target);
            const portsAndProtocols = getPortsAndProtocolsByLink(nodeLinkKey, isEgress);

            // if the edge is between two deployments in the same namespace
            if (inSameNamespace) {
                if (!nodeLinks[nodeLinkKey]) {
                    const classes = getClasses({
                        ...coreClasses,
                        ...directionalClasses,
                        // if the edge is in the same namespace, it's hovered when the source/target lines up
                        hovered:
                            hoveredEdge?.sourceNodeId === source &&
                            hoveredEdge?.targetNodeId === target,
                    });
                    nodeLinks[nodeLinkKey] = {
                        data: {
                            destNodeId,
                            destNodeNamespace,
                            destNodeName,
                            sourceNodeId,
                            sourceNodeName,
                            sourceNodeNamespace,
                            targetNodeId,
                            targetNodeName,
                            targetNodeNamespace,
                            portsAndProtocols,
                            traffic: isRelativeIngress
                                ? networkTraffic.INGRESS
                                : networkTraffic.EGRESS,
                            type: edgeTypes.NODE_TO_NODE_EDGE,
                            ...link,
                        },
                        classes,
                    };
                } else if (!nodeLinks[nodeLinkKey]?.data?.isBidirectional) {
                    // if this edge is already in the nodeLinks, it means it's going in the other direction
                    nodeLinks[nodeLinkKey].data.isBidirectional = true;
                    nodeLinks[nodeLinkKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[nodeLinkKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        // if the edge is bidirectional, it means the source/target is backwards if hovered
                        hovered:
                            hoveredEdge?.targetNodeId === source &&
                            hoveredEdge?.sourceNodeId === target,
                    });
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

                const isWithinSourceNS = highlightedNode?.parent === sourceNS;
                const isWithinTargetNS = highlightedNode?.parent === targetNS;

                const innerSourceEdgeKey = getSourceTargetKey(source, sourceNSSide);
                const innerTargetEdgeKey = getSourceTargetKey(targetNSSide, target);

                // if the hovered edge is a namespace edge, it hovers all the edges connected to the namespaces
                const isInnerNamespaceEdge =
                    isNamespaceEdge(hoveredEdge) &&
                    ((hoveredEdge?.sourceNodeNamespace === sourceNS &&
                        hoveredEdge?.targetNodeNamespace === targetNS) ||
                        (hoveredEdge?.sourceNodeNamespace === targetNS &&
                            hoveredEdge?.targetNodeNamespace === sourceNS));
                // if this edge is to the source namespace side, it's hovered when the source is the same
                const isInnerSourceEdgeHovered =
                    isInnerNamespaceEdge ||
                    (isNodeToNamespaceEdge(hoveredEdge) &&
                        (hoveredEdge?.sourceNodeId === source ||
                            hoveredEdge?.targetNodeId === source));
                // if this edge is to the target namespace side, it's hovered when the target is the same
                const isInnerTargetEdgeHovered =
                    isInnerNamespaceEdge ||
                    (isNodeToNamespaceEdge(hoveredEdge) &&
                        (hoveredEdge?.targetNodeId === target ||
                            hoveredEdge?.sourceNodeId === target));

                if (!nodeLinks[innerSourceEdgeKey]) {
                    // if the inner edge from source/target to namespace is in the same namespace as selected
                    const classes = getClasses({
                        ...coreClasses,
                        ...directionalClasses,
                        inner: true,
                        withinNS: isWithinSourceNS,
                        hovered: isInnerSourceEdgeHovered,
                    });
                    // Edge from source deployment to it's namespace edge
                    nodeLinks[innerSourceEdgeKey] = {
                        data: {
                            destNodeId,
                            destNodeName,
                            source,
                            target: sourceNSSide,
                            destNodeNamespace,
                            sourceNodeId,
                            sourceNodeName,
                            sourceNodeNamespace,
                            targetNodeId,
                            targetNodeName,
                            targetNodeNamespace,
                            portsAndProtocols,
                            isActive,
                            isAllowed,
                            isDisallowed,
                            traffic: isRelativeIngress
                                ? networkTraffic.INGRESS
                                : networkTraffic.EGRESS,
                            type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                        },
                        classes,
                    };
                } else if (
                    !nodeLinks[innerSourceEdgeKey].data.isBidirectional &&
                    !isWithinSourceNS
                ) {
                    // if this edge is already in the nodeLinks, it means it's going in the other direction
                    nodeLinks[innerSourceEdgeKey].data.isBidirectional = true;
                    nodeLinks[innerSourceEdgeKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[innerSourceEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        hovered: isInnerSourceEdgeHovered,
                    });
                }

                if (!nodeLinks[innerTargetEdgeKey]) {
                    const classes = getClasses({
                        ...coreClasses,
                        ...directionalClasses,
                        inner: true,
                        withinNS: isWithinTargetNS,
                        hovered: isInnerTargetEdgeHovered,
                    });

                    // Edge from namespace edge to target deployment
                    nodeLinks[innerTargetEdgeKey] = {
                        data: {
                            source: targetNSSide,
                            target,
                            destNodeId,
                            destNodeName,
                            destNodeNamespace,
                            sourceNodeId,
                            sourceNodeName,
                            sourceNodeNamespace,
                            targetNodeId,
                            targetNodeName,
                            targetNodeNamespace,
                            isActive,
                            isDisallowed,
                            portsAndProtocols,
                            type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                            traffic: isRelativeIngress
                                ? networkTraffic.INGRESS
                                : networkTraffic.EGRESS,
                        },
                        classes,
                    };
                } else if (
                    !nodeLinks[innerTargetEdgeKey].data.isBidirectional &&
                    !isWithinTargetNS
                ) {
                    // if this edge is already in the nodeLinks, it means it's going in the other direction
                    nodeLinks[innerTargetEdgeKey].data.isBidirectional = true;
                    nodeLinks[innerTargetEdgeKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[innerTargetEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        hovered: isInnerTargetEdgeHovered,
                    });
                }
            }
        }
    });

    return Object.values(nodeLinks);
};

/**
 * Select out the entity representing external connections in the cluster
 *
 * @param   {!Object[]} data    list of "deployments", without the external entity filtered out
 * @param   {!Object} configObj config object of the current network graph state
 *                              that contains links, filterState, and nodeSideMap,
 *                              networkNodeMap, hoveredNode, and selectedNode
 *
 * @return  {!Object}
 */
export const getInternetNode = (data, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;

    const externalNode = data.find((datum) => datum?.entity?.type === EXTERNAL_ENTITIES_TYPE);

    if (!externalNode) {
        return null;
    }

    const { entity, ...datumProps } = externalNode;
    const entityData = networkNodeMap[entity.id];
    const edges = getEdgesFromNode(configObj);

    const externallyConnected =
        filterState === filterModes.all
            ? entityData.active.externallyConnected
            : externalNode.externallyConnected;

    const isSelected = !!(selectedNode?.id === entity.id);
    const isHovered = !!(hoveredNode?.id === entity.id);
    const isBackground = !(!selectedNode && !hoveredNode) && !isHovered && !isSelected;
    const isNonIsolated = isNonIsolatedNode(externalNode);
    const isDisallowed =
        filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
    const isExternallyConnected = externallyConnected && filterState !== filterModes.allowed;
    const classes = getClasses({
        active: externalNode.isActive,
        selected: isSelected,
        internet: true,
        disallowed: isDisallowed,
        hovered: isHovered,
        background: isBackground,
        nonIsolated: isNonIsolated,
        externallyConnected: isExternallyConnected,
    });

    return {
        data: {
            ...datumProps,
            ...entity,
            id: entity?.id,
            name: 'External\nEntities \u2b08',
            active: false,
            type: EXTERNAL_ENTITIES_TYPE,
            parent: null,
        },
        classes,
    };
};

/**
 * Helper function that returns a function to determine whether a given id is in the list
 *
 * @param {!string} entityId entity id to match on
 *
 * @returns {!Function}
 */
const findEntityId = (entityId) => {
    return ({ data }) => data.targetNodeId === entityId || data.sourceNodeId === entityId;
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
export const getDeploymentList = (filteredData, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap, featureFlags } = configObj;
    const deploymentList = filteredData.map((datum) => {
        const { entity, ...datumProps } = datum;
        const { deployment, ...entityProps } = entity;
        const { namespace, ...deploymentProps } = deployment;

        const entityData = networkNodeMap[entity.id];

        const edges = getEdgesFromNode(configObj);
        const showExternalSources = isBackendFeatureFlagEnabled(
            featureFlags,
            knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
            false
        );

        const externallyConnected =
            filterState === filterModes.all
                ? entityData.active.externallyConnected
                : datum.externallyConnected;

        const isSelected = !!(selectedNode?.id === entity.id);
        const isHovered = !!(hoveredNode?.id === entity.id);
        const isAdjacentToSelected = selectedNode?.edges?.find(findEntityId(entity.id));
        const isAdjacentToHovered = hoveredNode?.edges?.find(findEntityId(entity.id));
        const isAdjacent =
            (!isHovered && isAdjacentToHovered) || (!isSelected && isAdjacentToSelected);
        const isBackground =
            !isAdjacent && (selectedNode || hoveredNode) && !isHovered && !isSelected;
        const isNonIsolated = isNonIsolatedNode(datum);
        const isDisallowed =
            filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
        const isExternallyConnected =
            showExternalSources && externallyConnected && filterState !== filterModes.allowed;
        const classes = getClasses({
            active: datum.isActive,
            selected: isSelected,
            deployment: true,
            disallowed: isDisallowed,
            hovered: isHovered,
            background: isBackground,
            nonIsolated: isNonIsolated,
            externallyConnected: isExternallyConnected,
        });

        let ingress = [];
        let egress = [];
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
    return [...getNamespaceEdges(configObj), ...getEdgesFromNode(configObj)];
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
export const getNamespaceList = (
    filteredData,
    deploymentList,
    { hoveredNode, selectedNode },
    cluster
) => {
    const activeNamespaceList = getActiveNamespaceList(filteredData, deploymentList);
    const highlightedNamespaces = {};
    const hoveredNodeEdges = hoveredNode?.edges;
    const selectedNodeEdges = selectedNode?.edges;
    const namespaceList = uniq(
        filteredData.map(({ entity }) => {
            const { namespace } = entity.deployment;
            if (!highlightedNamespaces[namespace]) {
                highlightedNamespaces[namespace] =
                    hoveredNodeEdges?.find(findEntityId(entity.id)) ||
                    selectedNodeEdges?.find(findEntityId(entity.id));
            }
            return namespace;
        })
    ).map((namespace) => {
        const isActive = activeNamespaceList.includes(namespace);
        const isHovered = hoveredNode?.id === namespace || hoveredNode?.parent === namespace;
        const isSelected = selectedNode?.id === namespace || selectedNode?.parent === namespace;
        const isAdjacent = highlightedNamespaces[namespace];
        const isBackground =
            !isAdjacent && (selectedNode || hoveredNode) && !isHovered && !isSelected;
        const classes = getClasses({
            nsGroup: true,
            nsActive: isActive,
            nsSelected: isSelected,
            nsHovered: isAdjacent || isHovered,
            background: isBackground,
        });

        return {
            data: {
                id: namespace,
                name: `${isActive ? '\ue901' : ''} ${namespace}`,
                active: isActive,
                type: entityTypes.NAMESPACE,
                parent: cluster,
            },
            classes,
        };
    });
    return namespaceList;
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
 * @param {!Object} networkNodeMap map of nodes by nodeId
 * @returns {!Object[]}
 */
const getActiveNetworkPolicyNodes = (networkNodeMap) => {
    const nodes = [];
    Object.keys(networkNodeMap).forEach((nodeId) => {
        const { active: activeNode, allowed: allowedNode } = networkNodeMap[nodeId];
        const node = { ...activeNode };
        if (allowedNode) {
            node.policyIds = flatMap(allowedNode.policyIds);
        }
        nodes.push(node);
    });
    return nodes;
};

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object} networkNodeMap map of nodes by nodeId
 * @param {string} filterState current filter state of the network graph
 * @returns {!Object[]}
 */
export const getFilteredNodes = (networkNodeMap, filterState) => {
    const activeNodes = [];
    const allowedNodes = [];
    Object.keys(networkNodeMap).forEach((id) => {
        if (networkNodeMap[id].active) {
            activeNodes.push(networkNodeMap[id].active);
        }
        if (networkNodeMap[id].allowed) {
            allowedNodes.push(networkNodeMap[id].allowed);
        }
    });
    if (filterState !== filterModes.active) {
        return allowedNodes;
    }

    // return as is
    if (!allowedNodes || !activeNodes) {
        return activeNodes;
    }

    return getActiveNetworkPolicyNodes(networkNodeMap);
};

/**
 * Grabs the deployment-to-deployment edges and filters based on the filter state
 *
 * @param {!Object[]} edges
 * @param {!Number} filterState
 * @returns {!Object[]}
 */
export function getNetworkFlows(edges, filterState) {
    if (!edges) {
        return [];
    }

    function getConnectionText(isActive) {
        let connection = '-';
        if (isActive) {
            connection = networkConnections.ACTIVE;
        } else {
            connection = networkConnections.ALLOWED;
        }
        return connection;
    }

    function DirectionalFlows() {
        let numIngressFlows = 0;
        let numEgressFlows = 0;
        return {
            incrementFlows: (traffic) => {
                if (
                    traffic === networkTraffic.INGRESS ||
                    traffic === networkTraffic.BIDIRECTIONAL
                ) {
                    numIngressFlows += 1;
                }
                if (traffic === networkTraffic.EGRESS || traffic === networkTraffic.BIDIRECTIONAL) {
                    numEgressFlows += 1;
                }
            },
            getNumIngressFlows: () => numIngressFlows,
            getNumEgressFlows: () => numEgressFlows,
        };
    }

    let networkFlows;
    const directionalFlows = new DirectionalFlows();
    const nodeMapping = edges.reduce(
        (
            acc,
            {
                data: {
                    destNodeId,
                    traffic,
                    destNodeName,
                    destNodeNamespace,
                    isActive,
                    portsAndProtocols,
                },
            }
        ) => {
            // don't double count edges that are divided because they're within different namespaces
            if (acc[destNodeId]) {
                return acc;
            }
            const connection = getConnectionText(isActive);
            directionalFlows.incrementFlows(traffic);
            return {
                ...acc,
                [destNodeId]: {
                    traffic,
                    deploymentId: destNodeId,
                    deploymentName: destNodeName,
                    namespace: destNodeNamespace,
                    connection,
                    portsAndProtocols,
                },
            };
        },
        {}
    );
    switch (filterState) {
        case filterModes.active:
            networkFlows = Object.values(nodeMapping).filter(
                (edge) => edge.connection === networkConnections.ACTIVE
            );
            break;
        case filterModes.allowed:
            networkFlows = Object.values(nodeMapping).filter(
                (edge) => edge.connection === networkConnections.ALLOWED
            );
            break;
        default:
            networkFlows = Object.values(nodeMapping);
    }
    const numIngressFlows = directionalFlows.getNumIngressFlows();
    const numEgressFlows = directionalFlows.getNumEgressFlows();
    return { networkFlows, numIngressFlows, numEgressFlows };
}

/**
 * Grabs either the ingress or egress ports and protocols from the network flows
 *
 * @param {!Object[]} networkFlows
 * @param {!String} traffic
 * @returns {!Object[]}
 */
function getPortsAndProtocolsByDirectionality(networkFlows, traffic) {
    if (!networkFlows) {
        return [];
    }
    return networkFlows.reduce((acc, networkFlow) => {
        return [
            ...acc,
            ...networkFlow.portsAndProtocols.filter((datum) => datum.traffic === traffic),
        ];
    }, []);
}

/**
 * Grabs either the ingress ports and protocols from the network flows
 *
 * @param {!Object[]} networkFlows
 * @returns {!Object[]}
 */
export function getIngressPortsAndProtocols(networkFlows) {
    return getPortsAndProtocolsByDirectionality(networkFlows, networkTraffic.INGRESS);
}

/**
 * Grabs either the egress ports and protocols from the network flows
 *
 * @param {!Object[]} networkFlows
 * @returns {!Object[]}
 */
export function getEgressPortsAndProtocols(networkFlows) {
    return getPortsAndProtocolsByDirectionality(networkFlows, networkTraffic.EGRESS);
}

/**
 * Determines if the node is hoverable (like deployment or external entities)
 *
 * @param {string} type the type of our graph node
 *
 * @return {boolean}
 */
export function getIsNodeHoverable(type) {
    return type === entityTypes.DEPLOYMENT || type === EXTERNAL_ENTITIES_TYPE;
}
