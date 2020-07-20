import { uniq, flatMap } from 'lodash';

import featureFlags from 'utils/featureFlags';
import entityTypes from 'constants/entityTypes';
import { filterModes } from 'Containers/Network/Graph/filterModes';

export const isNonIsolatedNode = (node) => node.nonIsolatedIngress && node.nonIsolatedEgress;

export const isDeployment = (node) => node && node.type === entityTypes.DEPLOYMENT;

export const isNamespace = (node) => node && node.type === entityTypes.NAMESPACE;

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object[]} nodes list of nodes
 * @returns {!Object[]}
 */
export const getLinks = (nodes, networkEdgeMap, networkNodeMap) => {
    const filteredLinks = [];

    nodes.forEach((node) => {
        if (!node.entity || node.entity.type !== 'DEPLOYMENT' || !networkEdgeMap) {
            return;
        }
        const { id: srcDeploymentId, deployment: srcDeployment } = node.entity;
        const sourceNS = srcDeployment && srcDeployment.namespace;

        const isActive = (key) => !!(networkEdgeMap[key] && networkEdgeMap[key].active);
        const isNonIsolated = (id) => !!(networkNodeMap[id] && networkNodeMap[id].nonIsolated);
        const isBetweenNonIsolated = (srcId, tgtId) => isNonIsolated(srcId) && isNonIsolated(tgtId);
        const isAllowed = (key, { source, target, targetNS }) =>
            sourceNS === 'stackrox' ||
            targetNS === 'stackrox' ||
            isBetweenNonIsolated(source, target) ||
            !!(networkEdgeMap[key] && networkEdgeMap[key].allowed);
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
                    !targetNode.entity ||
                    targetNode.entity.type !== 'DEPLOYMENT' ||
                    !targetNode.nonIsolatedIngress // nodes that are ingress-isolated have explicit incoming edges
                ) {
                    return;
                }

                const { id: tgtDeploymentId, deployment: tgtDeployment } = targetNode.entity;
                const targetNS = tgtDeployment && tgtDeployment.namespace;
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
            if (!tgtNode || !tgtNode.entity || tgtNode.entity.type !== 'DEPLOYMENT') {
                return;
            }
            const { id: tgtDeploymentId, deployment: tgtDeployment } = tgtNode.entity;
            const targetNS = tgtDeployment && tgtDeployment.namespace;
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
 * Iterates through a list of links and returns bundled edges between namespaces
 *
 * @param {string} nodeId nodeId
 * @param {!Object} configObj config object of the current network graph state
 *                            that contains links, filterState, and nodeSideMap
 * @returns {!Object[]} list of objects describing bundled edges between namespaces
 */
export const getNamespaceEdges = (nodeId, { links, filterState, nodeSideMap }) => {
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
    const activeLinkMap = {};
    const edgeBundleCountMap = {};
    filteredLinks.forEach(({ source, target, sourceNS, targetNS, isActive, isDisallowed }) => {
        const NSLinkKey = [sourceNS, targetNS].sort().join(delimiter);
        if (isActive) activeLinkMap[NSLinkKey] = true;
        if (isDisallowed) disallowedLinkMap[NSLinkKey] = true;

        const namespaceSourceTargetKey = [source, target].sort().join(delimiter);
        if (!sourceTargetMap[namespaceSourceTargetKey]) {
            sourceTargetMap[namespaceSourceTargetKey] = true;
            edgeBundleCountMap[NSLinkKey] = edgeBundleCountMap[NSLinkKey]
                ? edgeBundleCountMap[NSLinkKey] + 1
                : 1;
        }
    });

    return Object.keys(edgeBundleCountMap).map((key) => {
        const [sourceNS, targetNS] = key.split(delimiter);
        const count = edgeBundleCountMap[key];
        const isActive = activeLinkMap[key];
        const activeClass = filterState !== filterModes.allowed && isActive ? 'active' : '';
        const disallowedClass =
            filterState !== filterModes.allowed && isActive && disallowedLinkMap[key]
                ? 'disallowed'
                : '';
        const { source, target } = getSideMap(sourceNS, targetNS, nodeSideMap) || {
            source: sourceNS,
            target: targetNS,
        };

        return {
            data: {
                source,
                target,
                count,
            },
            classes: `namespace ${activeClass} ${disallowedClass}`,
        };
    });
};

/**
 * Iterates through a mapping of classes to boolean types to return a string of appended classes
 *
 * @param {!Object} map object containing className to boolean properties
 * @returns {!string}
 */
export const getClasses = (map) => {
    return Object.entries(map)
        .filter((entry) => entry[1])
        .map((entry) => entry[0])
        .join(' ');
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
    const edgeMap = {};
    const edges = [];
    const inAllowedState = filterState === filterModes.allowed;
    links.forEach((linkItem) => {
        const { source, sourceNS, sourceName, target, targetNS, targetName } = linkItem;
        const { isActive, isDisallowed, isBetweenNonIsolated } = linkItem;
        const isSourceNode = nodeId === source;
        const isTargetNode = nodeId === target;
        // destination node info needed for network flow tab
        const destNodeId = isSourceNode ? target : source;
        const destNodeNS = isSourceNode ? targetNS : sourceNS;
        const destNodeName = isSourceNode ? targetName : sourceName;
        if ((isSourceNode || isTargetNode) && (filterState !== filterModes.active || isActive)) {
            const classes = getClasses({
                active: !inAllowedState && isActive,
                // only hide edge when it's bw nonisolated and is not active
                nonIsolated: isBetweenNonIsolated && (!isActive || inAllowedState),
                // an edge is disallowed when it is active but is not allowed
                disallowed: !inAllowedState && isDisallowed,
                // if the currently hovered node is a target for this link (ingress)
                ingress: hoveredNode?.id === target || selectedNode?.id === target,
                // if the currently hovered node is a source for this link (egress)
                egress: hoveredNode?.id === source || selectedNode?.id === source,
            });
            const id = [source, target].sort().join('--');
            if (!edgeMap[id]) {
                // If same namespace, draw line between the two nodes
                if (sourceNS === targetNS) {
                    edges.push({
                        data: {
                            destNodeId,
                            destNodeNS,
                            destNodeName,
                            ...linkItem,
                        },
                        classes: `edge ${classes}`,
                    });
                } else {
                    // make sure both nodes have edges drawn to the nearest side of their NS
                    let sourceNSSide = sourceNS;
                    let targetNSSide = targetNS;
                    const sideMap = getSideMap(sourceNS, targetNS, nodeSideMap);
                    if (sideMap) {
                        sourceNSSide = sideMap.source;
                        targetNSSide = sideMap.target;
                    }

                    // Edge from source to it's namespace
                    edges.push({
                        data: {
                            source,
                            target: sourceNSSide,
                            isDisallowed,
                        },
                        classes: `edge inner ${classes}`,
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
                            isDisallowed,
                        },
                        classes: `edge inner ${classes}`,
                    });
                }
                edgeMap[id] = true;
            }
        }
    });
    return edges;
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

        const isSelected = !!(selectedNode && selectedNode.id === entity.id);
        const isHovered = !!(hoveredNode && hoveredNode.id === entity.id);
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

        let ingressCount = 0;
        let egressCount = 0;
        const entityData = networkNodeMap[entity.id];
        if (entityData) {
            const { ingressAllowed, ingressActive, egressAllowed, egressActive } = entityData;
            const ingressAllowedCount = ingressAllowed ? ingressAllowed.length : 0;
            const ingressActiveCount = ingressActive ? ingressActive.length : 0;
            const egressAllowedCount = egressAllowed ? egressAllowed.length : 0;
            const egressActiveCount = egressActive ? egressActive.length : 0;
            if (filterState === filterModes.allowed) {
                ingressCount = ingressAllowedCount;
                egressCount = egressAllowedCount;
            } else if (filterState === filterModes.active) {
                ingressCount = ingressActiveCount;
                egressCount = egressActiveCount;
            } else {
                egressCount = egressAllowedCount + egressActiveCount;
                ingressCount = ingressAllowedCount + ingressActiveCount;
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
                ingressCount,
                egressCount,
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
    const node = hoveredNode || selectedNode;
    let allEdges = getNamespaceEdges(node && node.id, configObj);
    if (node) {
        allEdges = allEdges.concat(getEdgesFromNode(node.id, configObj));
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
        const isHovered =
            hoveredNode && (hoveredNode.id === namespace || hoveredNode.parent === namespace);
        const isSelected =
            selectedNode && (selectedNode.id === namespace || selectedNode.parent === namespace);
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
            (allowedNode) =>
                allowedNode.entity && node.entity && allowedNode.entity.id === node.entity.id
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
