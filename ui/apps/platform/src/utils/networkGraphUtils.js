import uniq from 'lodash/uniq';
import flatMap from 'lodash/flatMap';

import { isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';
import entityTypes from 'constants/entityTypes';
import { networkTraffic, networkConnections, nodeTypes } from 'constants/networkGraph';
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

export const getIsExternalEntities = (type) => type === nodeTypes.EXTERNAL_ENTITIES;

export const getIsCIDRBlockNode = (type) => type === nodeTypes.CIDR_BLOCK;

export const isNamespaceEdge = (edge) => edge?.type === edgeTypes.NAMESPACE_EDGE;

export const isNodeToNodeEdge = (edge) => edge?.type === edgeTypes.NODE_TO_NODE_EDGE;

export const isNodeToNamespaceEdge = (edge) => edge?.type === edgeTypes.NODE_TO_NAMESPACE_EDGE;

export const getIsNodeExternal = (id, nodes) => {
    // TODO: There seems to be an inconsistency with the id for external entities. We should keep it as the "id"
    // provided by the backend rather than modifying it to be "External Entities"
    if (id === 'External Entities') {
        return true;
    }
    const node = nodes.find((datum) => {
        return datum?.entity?.id === id;
    });
    return node?.entity?.type === nodeTypes.CIDR_BLOCK;
};

/**
 * Gets the namespace value for a node
 *
 * @param {!Object} node
 *
 * @returns {!string}
 *
 */
export function getNodeNamespace(node) {
    const { deployment, id, type } = node.entity;
    const isExternalEntitiesNode = getIsExternalEntities(type);
    const isCIDRBlockNode = getIsCIDRBlockNode(type);
    if (isExternalEntitiesNode) {
        return 'External Entities';
    }
    if (isCIDRBlockNode) {
        return id;
    }
    return deployment.namespace;
}

/**
 * Gets the name value for a node
 *
 * @param {!Object} node
 *
 * @returns {!string}
 *
 */
export function getNodeName(node) {
    const { deployment, id, type } = node.entity;
    const isExternalEntitiesNode = getIsExternalEntities(type);
    const isCIDRBlockNode = getIsCIDRBlockNode(type);
    if (isExternalEntitiesNode) {
        return 'External Entities';
    }
    if (isCIDRBlockNode) {
        return id;
    }
    return deployment.name;
}

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
    unfilteredLinks = [],
    links = [],
    filterState,
    nodeSideMap,
    selectedNode,
    hoveredNode,
    hoveredEdge,
    networkNodeMap,
    featureFlags,
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

    const showExternalSources = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
        false
    );

    const linkArray = showExternalSources ? unfilteredLinks : links;

    const filteredLinks = linkArray.filter(
        ({ source, target, isActive, sourceNS, targetNS }) =>
            source &&
            target &&
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
    unfilteredLinks,
    links,
    nodes,
    nodeSideMap,
    hoveredNode,
    selectedNode,
    hoveredEdge,
    networkNodeMap,
    featureFlags,
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

    const showExternalSources = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
        false
    );

    const linkArray = showExternalSources ? unfilteredLinks : links;

    linkArray.forEach((link) => {
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
            const isSourceExternal = getIsNodeExternal(source, nodes);
            const isTargetExternal = getIsNodeExternal(target, nodes);
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
                let sourceParentSide = isSourceExternal ? source : sourceNS;
                let targetParentSide = isTargetExternal ? target : targetNS;
                const sideMap = getSideMap(sourceParentSide, targetParentSide, nodeSideMap);
                if (sideMap) {
                    sourceParentSide = sideMap.source;
                    targetParentSide = sideMap.target;
                }

                const isWithinSourceNS = highlightedNode?.parent === sourceNS;
                const isWithinTargetNS = highlightedNode?.parent === targetNS;

                const innerSourceEdgeKey = isSourceExternal
                    ? source
                    : getSourceTargetKey(source, sourceParentSide);
                const innerTargetEdgeKey = isTargetExternal
                    ? target
                    : getSourceTargetKey(targetParentSide, target);

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

                const innerSourceEdge = nodeLinks[innerSourceEdgeKey];
                const innerTargetEdge = nodeLinks[innerTargetEdgeKey];

                if (!innerSourceEdge && !isSourceExternal) {
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
                            source,
                            target: sourceParentSide,
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
                            isAllowed,
                            isDisallowed,
                            portsAndProtocols,
                            type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                            traffic: isRelativeIngress
                                ? networkTraffic.INGRESS
                                : networkTraffic.EGRESS,
                        },
                        classes,
                    };
                }

                if (!innerTargetEdge && !isTargetExternal) {
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
                            source: targetParentSide,
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
                            isAllowed,
                            isDisallowed,
                            portsAndProtocols,
                            type: edgeTypes.NODE_TO_NAMESPACE_EDGE,
                            traffic: isRelativeIngress
                                ? networkTraffic.INGRESS
                                : networkTraffic.EGRESS,
                        },
                        classes,
                    };
                }

                if (
                    innerSourceEdge &&
                    !innerSourceEdge?.data?.isBidirectional &&
                    !isWithinSourceNS &&
                    innerTargetEdgeKey &&
                    nodeLinks[innerTargetEdgeKey]
                ) {
                    // if this edge is already in the nodeLinks, it means it's going in the other direction
                    nodeLinks[innerSourceEdgeKey].data.isBidirectional = true;
                    nodeLinks[innerSourceEdgeKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[innerSourceEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        hovered: isInnerSourceEdgeHovered,
                    });

                    // we want to make sure the corresponding inner edge from the other namespace is also updated
                    nodeLinks[innerTargetEdgeKey].data.isBidirectional = true;
                    nodeLinks[innerTargetEdgeKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[innerTargetEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        hovered: isInnerTargetEdgeHovered,
                    });
                }

                if (innerTargetEdge && !innerTargetEdge.data.isBidirectional && !isWithinTargetNS) {
                    // if this edge is already in the nodeLinks, it means it's going in the other direction
                    nodeLinks[innerTargetEdgeKey].data.isBidirectional = true;
                    nodeLinks[innerTargetEdgeKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[innerTargetEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        hovered: isInnerTargetEdgeHovered,
                    });

                    // we want to make sure the corresponding inner edge from the other namespace is also updated
                    nodeLinks[innerSourceEdgeKey].data.isBidirectional = true;
                    nodeLinks[innerSourceEdgeKey].data.traffic = networkTraffic.BIDIRECTIONAL;
                    nodeLinks[innerSourceEdgeKey].classes = getClasses({
                        ...coreClasses,
                        bidirectional: true,
                        hovered: isInnerSourceEdgeHovered,
                    });
                }
            }
        }
    });

    return Object.values(nodeLinks);
};

/**
 * Create the cluster node for the network graph
 *
 * @param   {!String} clusterName
 *
 * @return  {!Object}
 */
export const getClusterNode = (clusterName) => {
    const clusterNode = {
        classes: 'cluster',
        data: {
            id: clusterName,
            name: clusterName,
            active: false,
            type: entityTypes.CLUSTER,
        },
    };
    return clusterNode;
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
export const getExternalEntitiesNode = (data, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;

    const externalNode = data.find((datum) => datum?.entity?.type === nodeTypes.EXTERNAL_ENTITIES);

    if (!externalNode) {
        return null;
    }

    const { entity, ...datumProps } = externalNode;
    const entityData = networkNodeMap[entity.id];
    const edges = getEdgesFromNode(configObj);

    const externallyConnected =
        // TODO: figure out how this should be handled in External Entity context
        filterState === filterModes.all
            ? entityData?.active?.externallyConnected
            : externalNode?.externallyConnected;

    const isSelected = !!(selectedNode?.id === entity.id);
    const isHovered = !!(hoveredNode?.id === entity.id);
    const isBackground = !(!selectedNode && !hoveredNode) && !isHovered && !isSelected;
    // DEPRECATED: const isNonIsolated = isNonIsolatedNode(externalNode);
    const isDisallowed =
        filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
    const isExternallyConnected = externallyConnected && filterState !== filterModes.allowed;
    const classes = getClasses({
        active: false, // externalNode.isActive,
        selected: isSelected,
        internet: true,
        disallowed: isDisallowed,
        hovered: isHovered,
        background: isBackground,
        nonIsolated: false,
        externallyConnected: isExternallyConnected,
    });

    return {
        data: {
            ...datumProps,
            ...entity,
            id: 'External Entities', // also needs to be `entity.id` in hover/select context
            name: 'External Entities',
            active: false,
            edges,
            type: nodeTypes.EXTERNAL_ENTITIES,
            parent: null,
        },
        classes,
    };
};

/**
 * Select out the entities representing external connections to CIDR blocks in the cluster
 *
 * @param   {!Object[]} data    list of "deployments", without the external entity filtered out
 * @param   {!Object} configObj config object of the current network graph state
 *                              that contains links, filterState, and nodeSideMap,
 *                              networkNodeMap, hoveredNode, and selectedNode
 *
 * @return  {!Object}
 */
export const getCIDRBlockNodes = (data, configObj = {}) => {
    const { hoveredNode, selectedNode, filterState, networkNodeMap } = configObj;

    const cidrBlocks = data.filter((datum) => datum?.entity?.type === nodeTypes.CIDR_BLOCK);

    if (cidrBlocks.length === 0) {
        return null;
    }

    const cidrBlockNodes = cidrBlocks.map((cidrBlock) => {
        const { entity, ...datumProps } = cidrBlock;
        const entityData = networkNodeMap[entity.id];
        const edges = getEdgesFromNode(configObj);

        const externallyConnected =
            filterState === filterModes.all
                ? entityData?.active?.externallyConnected
                : cidrBlock?.externallyConnected;

        const isSelected = !!(selectedNode?.id === entity.id);
        const isHovered = !!(hoveredNode?.id === entity.id);
        const isBackground = !(!selectedNode && !hoveredNode) && !isHovered && !isSelected;
        // DEPRECATED: const isNonIsolated = isNonIsolatedNode(externalNode);
        const isDisallowed =
            filterState !== filterModes.allowed && edges.some((edge) => edge.data.isDisallowed);
        const isExternallyConnected = externallyConnected && filterState !== filterModes.allowed;
        const classes = getClasses({
            active: false,
            selected: isSelected,
            cidrBlock: true,
            disallowed: isDisallowed,
            hovered: isHovered,
            background: isBackground,
            nonIsolated: false,
            externallyConnected: isExternallyConnected,
        });

        return {
            data: {
                ...datumProps,
                ...entity,
                id: entity.id,
                cidr: entity.externalSource.cidr,
                name: entity.externalSource.name,
                edges,
                active: false,
                type: nodeTypes.CIDR_BLOCK,
                parent: null,
            },
            classes,
        };
    });
    return cidrBlockNodes;
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

        const showExternalSources = isBackendFeatureFlagEnabled(
            featureFlags,
            knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
            false
        );

        // need to change edges to include external sources
        const edges = getEdgesFromNode(configObj);

        const { externallyConnected } = filterState === filterModes.all ? entityData.active : datum;

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
    const namespaceEdges = getNamespaceEdges(configObj);
    const edgesFromNodes = getEdgesFromNode(configObj);
    return [...namespaceEdges, ...edgesFromNodes];
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
                name: `${isActive ? '\ue901 ' : ''}${namespace}`,
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
 * Returns a list of edge nodes that are hidden cardinal direction edges
 *
 * @param {!String} name
 * @param {!String} classes

 * @returns {!Object[]}
 */
const sides = ['top', 'left', 'right', 'bottom'];

const createEdgeNodes = (id, classes) => {
    const edgeNodes = sides.reduce((acc, side) => {
        const node = {
            data: {
                id: `${id}_${side}`,
                parent: id,
                side,
            },
            classes,
        };
        return [...acc, node];
    }, []);
    return edgeNodes;
};

export const getEdgeNodes = (nodeList, classes) => {
    const totalEdgeNodes = nodeList.reduce((acc, node) => {
        const { id } = node.data;
        const edgeNodes = createEdgeNodes(id, classes);
        return [...acc, ...edgeNodes];
    }, []);
    return totalEdgeNodes;
};

/**
 * Returns a list of nodes that are hidden "namespace" cardinal direction edges
 *
 * @param {!Object[]} namespaceList list of namespaces
 *
 * @returns {!Object[]}
 */
export const getNamespaceEdgeNodes = (namespaces) => {
    const namespaceEdgeNodes = getEdgeNodes(namespaces, 'nsEdge');
    return namespaceEdgeNodes;
};

/**
 * Returns a list of nodes that are hidden "external entities" cardinal direction edges
 *
 * @param {!Object} externalEntitiesNode
 *
 * @returns {!Object[]}
 */
export const getExternalEntitiesEdgeNodes = (externalEntitiesNode) => {
    const externalEntitiesEdgeNodes = getEdgeNodes([externalEntitiesNode], 'externalEntitiesEdge');
    return externalEntitiesEdgeNodes;
};

/**
 * Returns a list of nodes that are hidden "external source" cardinal direction edges
 *
 * @param {!Object} cidrBlockNodes
 *
 * @returns {!Object[]}
 */
export const getCIDRBlockEdgeNodes = (cidrBlockNodes) => {
    const cidrBlockEdgeNodes = getEdgeNodes(cidrBlockNodes, 'cidrBlockEdge');
    return cidrBlockEdgeNodes;
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

function getConnectionText(filterState, isActive, isAllowed) {
    let connection = '-';
    const isActiveOrAll = filterState === filterModes.active || filterState === filterModes.all;
    const isAllowedOrAll = filterState === filterModes.allowed || filterState === filterModes.all;
    if (isActiveOrAll && isActive) {
        connection = networkConnections.ACTIVE;
    } else if (isAllowedOrAll && isAllowed) {
        connection = networkConnections.ALLOWED;
    }
    return connection;
}

function DirectionalFlows() {
    let numIngressFlows = 0;
    let numEgressFlows = 0;
    return {
        incrementFlows: (traffic) => {
            if (traffic === networkTraffic.INGRESS || traffic === networkTraffic.BIDIRECTIONAL) {
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
                    isAllowed,
                    portsAndProtocols,
                },
            }
        ) => {
            // don't double count edges that are divided because they're within different namespaces
            if (acc[destNodeId]) {
                return acc;
            }
            const connection = getConnectionText(filterState, isActive, isAllowed);
            directionalFlows.incrementFlows(traffic);
            return {
                ...acc,
                [destNodeId]: {
                    traffic,
                    deploymentId: destNodeId,
                    entityName: destNodeName,
                    namespace: destNodeNamespace === 'External Entities' ? '-' : destNodeNamespace,
                    type: destNodeNamespace === 'External Entities' ? 'external' : 'deployment',
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
    return (
        type === entityTypes.DEPLOYMENT ||
        type === nodeTypes.EXTERNAL_ENTITIES ||
        type === nodeTypes.CIDR_BLOCK
    );
}
