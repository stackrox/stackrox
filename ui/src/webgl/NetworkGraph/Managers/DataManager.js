import * as constants from 'constants/networkGraph';
import {
    getLinks,
    getLinksBetweenNamespaces,
    getBidirectionalLinks
} from 'utils/networkGraphUtils';

const DataManager = canvas => {
    const { clientWidth, clientHeight } = canvas;

    let simulation;

    let nodes = [];
    let links = [];
    let namespaces = [];
    let namespaceLinks = [];

    function setUpForceLayout(callback) {
        const deploymentNodes = nodes.filter(n => n.deploymentId);

        if (callback) {
            callback({
                nodes: deploymentNodes,
                links,
                namespaces,
                clientWidth,
                clientHeight,
                constants
            });
        }
    }

    function setNodes(_nodes) {
        nodes = _nodes;
    }

    function setLinks(_links) {
        links = _links;
    }

    function setNamespaces(_namespaces) {
        namespaces = _namespaces;
    }

    function getNamespaces(dataNodes) {
        const namespacesMapping = {};

        dataNodes.forEach(node => {
            if (!node.entity || node.entity.type !== 'DEPLOYMENT') {
                return;
            }
            const {
                deployment: { namespace }
            } = node.entity;
            let namespaceProperties = namespacesMapping[namespace];
            if (!namespaceProperties) {
                namespaceProperties = {
                    namespace,
                    internetAccess: false,
                    nodes: []
                };
                namespacesMapping[namespace] = namespaceProperties;
            }
            if (node.internetAccess) {
                namespaceProperties.internetAccess = true;
            }
            namespaceProperties.nodes.push(node);
        });

        return Object.values(namespacesMapping);
    }

    function enrichNodes(dataNodes) {
        const namespacesMapping = {};

        const enrichedNodes = dataNodes.map(dataNode => {
            const node = { ...dataNode };

            if (dataNode.entity.type !== 'DEPLOYMENT') {
                return node;
            }

            const {
                id: deploymentId,
                deployment: { namespace, name: deploymentName }
            } = dataNode.entity;
            node.radius = constants.NODE_RADIUS;
            node.deploymentId = deploymentId;
            node.deploymentName = deploymentName;
            node.namespace = namespace;

            // set centroid
            if (!namespacesMapping[namespace]) {
                node.centroid = true;
                namespacesMapping[namespace] = node;
            }
            if (node.internetAccess) {
                namespacesMapping[namespace] = node;
            }

            return node;
        });

        return enrichedNodes;
    }

    function getNamespaceLinks(dataNodes, networkFlowMapping) {
        const linksBetweenNamespaces = getLinksBetweenNamespaces(dataNodes, networkFlowMapping);
        return getBidirectionalLinks(linksBetweenNamespaces);
    }

    function getData() {
        return {
            nodes,
            links,
            namespaces,
            namespaceLinks
        };
    }

    function setData(data) {
        nodes = enrichNodes(data.nodes);
        links = getLinks(nodes, data.networkFlowMapping);
        namespaces = getNamespaces(nodes);
        namespaceLinks = getNamespaceLinks(nodes, data.networkFlowMapping);
        simulation = setUpForceLayout(data.postMessageCallback);
    }

    return {
        setUpForceLayout,
        simulation,
        getData,
        setData,
        setNodes,
        setLinks,
        setNamespaces
    };
};

export default DataManager;
