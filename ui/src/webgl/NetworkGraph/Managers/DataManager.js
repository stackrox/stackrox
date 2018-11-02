import * as d3 from 'd3';
import * as constants from 'constants/networkGraph';
import {
    forceCollide,
    forceCluster,
    getLinksInSameNamespace,
    getLinksBetweenNamespaces,
    getBidirectionalLinks
} from 'utils/networkGraphUtils';

const DataManager = canvas => {
    const { clientWidth, clientHeight } = canvas;

    let simulation;
    let simulationRunning = true;

    let nodes = [];
    let links = [];
    let namespaces = [];
    let namespaceLinks = [];

    function setUpForceLayout() {
        simulationRunning = true;

        const forceSimulation = d3
            .forceSimulation()
            .nodes(nodes, d => d.deploymentId)
            .force(
                'link',
                d3
                    .forceLink(links)
                    .id(d => d.deploymentId)
                    .strength(0)
            )
            .force('charge', d3.forceManyBody())
            .force('center', d3.forceCenter(clientWidth / 2, clientHeight / 2))
            .force('collide', forceCollide(nodes))
            .force(
                'cluster',
                forceCluster().strength(constants.FORCE_CONFIG.FORCE_CLUSTER_STRENGTH)
            )
            .alpha(1)
            .stop();

        // create static force layout by calculating ticks beforehand
        let i = 0;
        const x = nodes.length * 5;
        while (i < x) {
            forceSimulation.tick();
            i += 1;
        }

        simulationRunning = false;

        return forceSimulation;
    }

    function getNamespaces(dataNodes) {
        const namespacesMapping = {};
        let foundNamespaces = [];

        dataNodes.forEach(node => {
            if (!namespacesMapping[node.namespace] || node.internetAccess) {
                const namespace = {
                    namespace: node.namespace,
                    internetAccess: node.internetAccess
                };
                namespacesMapping[node.namespace] = namespace;
            }
        });

        foundNamespaces = Object.values(namespacesMapping);

        return foundNamespaces.map(namespace => ({
            ...namespace,
            nodes: dataNodes.filter(node => node.namespace === namespace.namespace)
        }));
    }

    function enrichNodes(dataNodes) {
        const namespacesMapping = {};

        const enrichedNodes = dataNodes.map(dataNode => {
            const node = { ...dataNode };
            node.radius = constants.NODE_RADIUS;

            // set centroid
            if (!namespacesMapping[node.namespace]) {
                node.centroid = true;
                namespacesMapping[node.namespace] = node;
            }
            if (node.internetAccess) {
                namespacesMapping[node.namespace] = node;
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
        links = getLinksInSameNamespace(nodes, data.networkFlowMapping);
        namespaces = getNamespaces(nodes);
        namespaceLinks = getNamespaceLinks(nodes, data.networkFlowMapping);
        simulation = setUpForceLayout();
    }

    function isSimulationRunning() {
        return simulationRunning;
    }

    return {
        simulation,
        getData,
        setData,
        isSimulationRunning
    };
};

export default DataManager;
