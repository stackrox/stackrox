import { nodeTypes } from 'constants/networkGraph';
import { filterModes } from 'constants/networkFilterModes';
import {
    getIsNodeHoverable,
    getSourceTargetKey,
    getNodeNamespace,
    getNodeName,
} from 'utils/networkGraphUtils';

/**
 * Iterates through a list of nodes and returns only links in the same namespace
 *
 * @param {!Object[]} nodes list of nodes
 * @param {!Object} networkEdgeMap map of edges in the graph by srcId--tgtId key
 * @param {!Object} networkNodeMap map of nodes in the graph by nodeId
 * @param {!number} filterState current filter state
 * @returns {!Object[]}
 */
export const getLinks = (nodes, networkEdgeMap, networkNodeMap, filterState) => {
    const filteredLinks = [];

    const isActive = (edgeKey) => !!networkEdgeMap[edgeKey]?.active;
    const isNonIsolated = (nodeId) => !!networkNodeMap[nodeId]?.nonIsolated;
    const isBetweenNonIsolated = (sourceId, targetId) =>
        isNonIsolated(sourceId) && isNonIsolated(targetId);
    const isAllowed = (edgeKey, { source, target, targetNS, sourceNS }) =>
        sourceNS === 'stackrox' ||
        targetNS === 'stackrox' ||
        isBetweenNonIsolated(source, target) ||
        !!networkEdgeMap[edgeKey]?.allowed;
    // Historical note: isDisallowed was added in rox#2070 and then disabled in rox#2747
    /*
    const isDisallowed = (edgeKey, link) =>
        UIfeatureFlags.SHOW_DISALLOWED_CONNECTIONS &&
        isActive(edgeKey) &&
        !isAllowed(edgeKey, link);
    */

    nodes.forEach((node) => {
        const isHoverable = getIsNodeHoverable(node?.entity?.type);
        if (!isHoverable || !networkEdgeMap) {
            return;
        }

        const { id: sourceEntityId, type: sourceNodeType } = node.entity;
        const sourceNS = getNodeNamespace(node);
        const sourceName = getNodeName(node);
        const isSourceExternal =
            sourceNodeType === nodeTypes.EXTERNAL_ENTITIES ||
            sourceNodeType === nodeTypes.CIDR_BLOCK;

        // For nodes that are egress non-isolated, add outgoing edges to ingress non-isolated nodes, as long as the pair
        // of nodes is not fully non-isolated. This is a compromise to make the non-isolation highlight only apply in
        // the case when there are neither ingress nor egress policies (the data sent from the backend is optimized to
        // treat both phenomena separately and omit edges from a egress non-isolated to an ingress non-isolated
        // deployment, but that would be too confusing in the UI).
        if (filterState !== filterModes.active) {
            if (node.nonIsolatedEgress) {
                nodes.forEach((targetNode) => {
                    if (Object.is(node, targetNode)) {
                        return;
                    }

                    const { id: targetEntityId, type: targetNodeType } = targetNode.entity;
                    const targetNS = getNodeNamespace(targetNode);
                    const targetName = getNodeName(targetNode);
                    const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);

                    if (
                        !targetNode.nonIsolatedIngress // nodes that are ingress-isolated have explicit incoming edges
                    ) {
                        return;
                    }

                    if (!node.queryMatch && !targetNode.queryMatch) {
                        return;
                    }

                    const isTargetExternal =
                        targetNodeType === nodeTypes.EXTERNAL_ENTITIES ||
                        targetNodeType === nodeTypes.CIDR_BLOCK;

                    const link = {
                        source: sourceEntityId,
                        target: targetEntityId,
                        sourceName,
                        targetName,
                        sourceNS,
                        targetNS,
                        sourceType: sourceNodeType,
                        targetType: targetNodeType,
                        isExternal: isSourceExternal || isTargetExternal,
                    };

                    link.isActive = isActive(edgeKey);
                    link.isBetweenNonIsolated = isBetweenNonIsolated(
                        sourceEntityId,
                        targetEntityId
                    );
                    link.isAllowed = isAllowed(edgeKey, link);

                    // Do not draw implicit links between fully non-isolated nodes unless the connection is active.
                    const isImplicit = node.nonIsolatedIngress && targetNode.nonIsolatedEgress;
                    const isCurrentlyActive =
                        (filterState === filterModes.active || filterState === filterModes.all) &&
                        link.isActive;
                    if (!isImplicit || isCurrentlyActive) {
                        filteredLinks.push(link);
                    }
                });
            }
        }

        Object.keys(node.outEdges).forEach((targetNodeId) => {
            if (networkNodeMap[targetNodeId]) {
                const targetNode =
                    networkNodeMap[targetNodeId].active || networkNodeMap[targetNodeId].allowed;
                const { id: targetEntityId, type: targetNodeType } = targetNode.entity;
                const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);
                const targetNS = getNodeNamespace(targetNode);
                const targetName = getNodeName(targetNode);
                const isTargetExternal =
                    targetNodeType === nodeTypes.EXTERNAL_ENTITIES ||
                    targetNodeType === nodeTypes.CIDR_BLOCK;

                const link = {
                    source: sourceEntityId,
                    target: targetEntityId,
                    sourceName,
                    targetName,
                    sourceNS,
                    targetNS,
                    sourceType: sourceNodeType,
                    targetType: targetNodeType,
                    isExternal: isSourceExternal || isTargetExternal,
                };

                link.isActive = isActive(edgeKey);
                link.isBetweenNonIsolated = isBetweenNonIsolated(sourceEntityId, targetEntityId);
                link.isAllowed = isAllowed(edgeKey, link);

                filteredLinks.push(link);
            }
        });
    });

    // Remove links that do not have a corresponding node. A link with a `source` or `target` that
    // does not match an entity id of a node in the filtered node set will cause Cytoscape to
    // crash during rendering. This crash may be immediate, or when a user hovers over an offending
    // node in the visualization.
    const nodeIds = new Set(nodes.map((n) => n.entity.id));
    return filteredLinks.filter(({ source, target }) => nodeIds.has(source) && nodeIds.has(target));
};

/**
 * get links that are NOT external entities
 *
 * @param   {!Object[]}  links  edges betweem nodes
 *
 * @return  {!Object[]}
 */
export const getFilteredLinks = (links) => {
    const filteredLinks = [];
    links.forEach((link) => {
        if (link.sourceType !== 'INTERNET' && link.targetType !== 'INTERNET') {
            filteredLinks.push(link);
        }
    });
    return filteredLinks;
};
