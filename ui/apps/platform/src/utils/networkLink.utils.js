import entityTypes from 'constants/entityTypes';
import { filterModes } from 'constants/networkFilterModes';
import { nodeTypes } from 'constants/networkGraph';
import { UIfeatureFlags, isBackendFeatureFlagEnabled, knownBackendFlags } from 'utils/featureFlags';
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
 * @param {!string} filterState current filter state
 * @param {!string[]} featureFlags featureFlags
 * @returns {!Object[]}
 */
export const getLinks = (nodes, networkEdgeMap, networkNodeMap, filterState, featureFlags) => {
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

    const showExternalSources = isBackendFeatureFlagEnabled(
        featureFlags,
        knownBackendFlags.ROX_NETWORK_GRAPH_EXTERNAL_SRCS,
        false
    );

    nodes.forEach((node) => {
        const isHoverable = getIsNodeHoverable(node?.entity?.type);
        if (!isHoverable || !networkEdgeMap) {
            return;
        }

        const { id: sourceEntityId, deployment: sourceDeployment } = node.entity;
        const sourceNS = getNodeNamespace(node);
        const sourceName = getNodeName(node);

        // For nodes that are egress non-isolated, add outgoing edges to ingress non-isolated nodes, as long as the pair
        // of nodes is not fully non-isolated. This is a compromise to make the non-isolation highlight only apply in
        // the case when there are neither ingress nor egress policies (the data sent from the backend is optimized to
        // treat both phenomena separately and omit edges from a egress non-isolated to an ingress non-isolated
        // deployment, but that would be too confusing in the UI).
        if (node.nonIsolatedEgress) {
            nodes.forEach((targetNode) => {
                if (Object.is(node, targetNode)) {
                    return;
                }

                if (targetNode?.entity?.type === nodeTypes.EXTERNAL_ENTITIES) {
                    const link = {
                        source: sourceEntityId,
                        target: 'External Entities',
                        sourceName: sourceDeployment?.name,
                        targetName: 'External Entities',
                        sourceNS,
                        targetNS: 'External Entities',
                    };

                    filteredLinks.push(link);
                    return;
                }

                if (
                    targetNode?.entity?.type !== entityTypes.DEPLOYMENT ||
                    !targetNode.nonIsolatedIngress // nodes that are ingress-isolated have explicit incoming edges
                ) {
                    return;
                }

                const { deployment: targetDeployment } = targetNode.entity;
                const targetEntityId = targetDeployment?.id;
                const targetNS = getNodeNamespace(targetNode);
                const targetName = getNodeName(targetNode);
                const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);

                const link = {
                    source: sourceEntityId,
                    target: targetEntityId,
                    sourceName,
                    targetName,
                    sourceNS,
                    targetNS,
                };

                link.isActive = isActive(edgeKey);
                link.isBetweenNonIsolated = isBetweenNonIsolated(sourceEntityId, targetEntityId);
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

        Object.keys(node.outEdges).forEach((targetNodeId) => {
            const targetNode = networkNodeMap[targetNodeId].active;
            const { id: targetId, type: targetNodeType } = targetNode.entity;
            const targetEntityId = targetNodeType === 'INTERNET' ? 'External Entities' : targetId;
            if (targetNode?.entity?.type !== entityTypes.DEPLOYMENT && !showExternalSources) {
                return;
            }
            const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);
            const targetNS = getNodeNamespace(targetNode);
            const targetName = getNodeName(targetNode);

            const link = {
                source: sourceEntityId,
                target: targetEntityId,
                sourceName,
                targetName,
                sourceNS,
                targetNS,
            };

            link.isActive = isActive(edgeKey);
            link.isBetweenNonIsolated = isBetweenNonIsolated(sourceEntityId, targetEntityId);
            link.isAllowed = isAllowed(edgeKey, link);
            link.isDisallowed = isDisallowed(edgeKey, link);

            filteredLinks.push(link);
            filteredEdgeHashTable[edgeKey] = true;
        });

        // if in the All filter state, merge active outEdges for each node from networkNodeMap so
        // we include disallowed edges that are not part of the allowed node set
        if (filterState === filterModes.all) {
            // TODO: add active edges from External Entity types of nodes back in
            Object.keys(networkNodeMap[sourceEntityId]?.active?.outEdges || []).forEach(
                (targetNodeId) => {
                    const targetNode = networkNodeMap[targetNodeId];
                    const isExternal = showExternalSources && !targetNode?.entity;
                    const { id: targetId, deployment: targetDeployment, type: targetNodeType } =
                        targetNode.entity || {};

                    const targetEntityId =
                        targetNodeType === 'INTERNET' ? 'External Entities' : targetId;

                    const targetNS = isExternal ? 'External Entities' : targetDeployment?.namespace;
                    const targetName = isExternal ? 'External Entities' : targetDeployment?.name;
                    const edgeKey = getSourceTargetKey(sourceEntityId, targetEntityId);
                    // to prevent double counting if an edge is also allowed and active
                    if (!filteredEdgeHashTable[edgeKey]) {
                        const link = {
                            source: sourceEntityId,
                            target: targetEntityId,
                            sourceName,
                            targetName,
                            sourceNS,
                            targetNS,
                        };

                        link.isActive = isActive(edgeKey);
                        link.isBetweenNonIsolated = isBetweenNonIsolated(
                            sourceEntityId,
                            targetEntityId
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
 * get links that are NOT external entities
 *
 * @param   {!Object[]}  links  edges betweem nodes
 *
 * @return  {!Object[]}
 */
export const getFilteredLinks = (links) => {
    const filteredLinks = [];
    links.forEach((link) => {
        if (link.targetName !== 'External Entities') {
            filteredLinks.push(link);
        }
    });
    return filteredLinks;
};
