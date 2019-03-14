import * as constants from 'constants/networkGraph';
import { getLinks } from 'utils/networkGraphUtils';

const DataManager = canvas => {
    const { clientWidth, clientHeight } = canvas;

    let links = [];
    let namespaces = [];

    function getNodesFromNamespaces() {
        return namespaces.reduce((acc, curr) => [...acc, ...curr.nodes], []);
    }

    function setUpForceLayout(worker) {
        const nodes = getNodesFromNamespaces();
        const deploymentNodes = nodes.filter(n => n.deploymentId);
        if (worker && worker.postMessage) {
            worker.postMessage({
                nodes: deploymentNodes,
                links,
                namespaces,
                clientWidth,
                clientHeight,
                constants
            });
        }
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

    function getData() {
        const nodes = getNodesFromNamespaces();
        return {
            nodes,
            links,
            namespaces
        };
    }

    function setData(data) {
        const nodes = enrichNodes(data.nodes);
        links = getLinks(nodes, data.networkFlowMapping);
        namespaces = getNamespaces(nodes);
        setUpForceLayout(data.worker);
    }

    function setLinks(_links) {
        links = _links;
    }

    function setNamespaces(_namespaces) {
        namespaces = _namespaces;
    }

    return {
        setUpForceLayout,
        getData,
        setData,
        setLinks,
        setNamespaces
    };
};

export default DataManager;
