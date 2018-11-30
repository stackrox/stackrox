import * as d3 from 'd3';
import * as constants from 'constants/networkGraph';
import {
    forceCollide,
    forceCluster,
    getLinks,
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

        const deploymentNodes = nodes.filter(n => n.deploymentId);
        const forceSimulation = d3
            .forceSimulation()
            .nodes(deploymentNodes, d => d.deploymentId)
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
        const x = deploymentNodes.length * 5;
        while (i < x) {
            forceSimulation.tick();
            i += 1;
        }

        simulationRunning = false;

        return forceSimulation;
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
