import React, { Component } from 'react';
import PropTypes from 'prop-types';

import * as THREE from 'three';
import * as d3 from 'd3';
import threeOrbitControls from 'three-orbit-controls';
import { forceCluster, getLinksInSameNamespace } from 'utils/networkGraphUtils/networkGraphUtils';
import * as constants from 'utils/networkGraphUtils/networkGraphConstants';

const OrbitControls = threeOrbitControls(THREE);

let nodes = [];
let links = [];
let namespaces = [];
let simulation = null;

class NetworkGraph extends Component {
    static propTypes = {
        nodes: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        onNodeClick: PropTypes.func.isRequired,
        updateKey: PropTypes.number.isRequired
    };

    componentDidMount() {
        this.setUpScene();
    }

    shouldComponentUpdate(nextProps) {
        if (!simulation || nextProps.updateKey !== this.props.updateKey) {
            // Clear the canvas
            this.clear();

            // Create objects for the scene
            nodes = this.setUpNodes(nextProps.nodes);
            namespaces = this.setUpNamespaces(nodes);
            links = this.setUpLinks(nextProps.nodes, nextProps.links);

            this.setUpForceSimulation();

            this.animate();
        }

        return false;
    }

    onGraphClick = ({ layerX: x, layerY: y }) => {
        const intersects = this.getIntersectingObjects(x, y);

        if (intersects.length) {
            const node = nodes.find(n => n.geometry.uuid === intersects[0].object.geometry.uuid);
            this.props.onNodeClick(node);
        }
    };

    onMouseMove = ({ layerX: x, layerY: y }) => {
        const intersects = this.getIntersectingObjects(x, y);

        if (intersects.length) {
            this.networkGraph.classList.add('cursor-pointer');
        } else {
            this.networkGraph.classList.remove('cursor-pointer');
        }
    };

    getIntersectingObjects = (x, y) => {
        const { clientWidth, clientHeight } = this.renderer.domElement;
        this.mouse.x = x / clientWidth * 2 - 1;
        this.mouse.y = -(y / clientHeight) * 2 + 1;

        // update the ray caster with the camera and mouse position
        this.raycaster.setFromCamera(this.mouse, this.camera);

        // calculate objects in the scene that intersect the ray caster
        const intersects = this.raycaster.intersectObjects(this.scene.children);

        return intersects;
    };

    setUpScene = () => {
        const { clientWidth, clientHeight } = this.networkGraph;

        this.raycaster = new THREE.Raycaster();
        this.mouse = new THREE.Vector2();

        // setup the scene
        this.scene = new THREE.Scene();

        // setup the camera
        this.camera = new THREE.OrthographicCamera(
            0,
            clientWidth,
            clientHeight,
            0,
            constants.MIN_ZOOM,
            constants.MAX_ZOOM
        );
        this.camera.position.z = constants.MIN_ZOOM;

        // setup the renderer
        this.renderer = new THREE.WebGLRenderer(constants.RENDERER_CONFIG);
        this.renderer.setSize(clientWidth, clientHeight);
        this.renderer.setPixelRatio(window.devicePixelRatio);

        // setup the orbit controls used for panning+zooming
        this.controls = new OrbitControls(this.camera, this.renderer.domElement);
        Object.assign(this.controls, constants.ORBIT_CONTROLS_CONFIG);

        // setup the canvas for the network graph
        this.networkGraph.appendChild(this.renderer.domElement);

        // setup event listeners
        this.renderer.domElement.addEventListener('click', this.onGraphClick, false);
        this.renderer.domElement.addEventListener('mousemove', this.onMouseMove, false);
    };

    setUpForceSimulation = () => {
        const { clientWidth, clientHeight } = this.networkGraph;

        simulation = d3
            .forceSimulation()
            .nodes(nodes, d => d.id)
            .force(
                'link',
                d3
                    .forceLink(links)
                    .id(d => d.id)
                    .strength(0)
            )
            .force('charge', d3.forceManyBody())
            .force('center', d3.forceCenter(clientWidth / 2, clientHeight / 2))
            .force(
                'collide',
                d3
                    .forceCollide()
                    .radius(d => d.radius + constants.FORCE_CONFIG.FORCE_COLLISION_RADIUS_OFFSET)
            )
            .force(
                'cluster',
                forceCluster(namespaces).strength(constants.FORCE_CONFIG.FORCE_CLUSTER_STRENGTH)
            )
            .alpha(1)
            .stop();

        // create static force layout by calculating ticks beforehand
        let i = 0;
        const x = nodes.length * 10;
        while (i < x) {
            simulation.tick();
            i += 1;
        }

        // restart force simulation
        simulation.restart();
    };

    setUpNodes = propNodes => {
        const newNodes = [];
        propNodes.forEach(propNode => {
            const node = { ...propNode };

            node.geometry = new THREE.CircleBufferGeometry(5, 32);
            node.radius = 1;
            node.material = new THREE.MeshBasicMaterial({
                color: 0x5a6fd9
            });
            node.circle = new THREE.Mesh(node.geometry, node.material);
            this.scene.add(node.circle);

            newNodes.push(node);
        });
        return newNodes;
    };

    setUpNamespaces = propNodes => {
        const namespacesMapping = {};
        propNodes.forEach(propNode => {
            if (!namespacesMapping[propNode.namespace] || propNode.internetAccess)
                namespacesMapping[propNode.namespace] = propNode;
        });
        return Object.values(namespacesMapping);
    };

    setUpLinks = (propNodes, propLinks) => {
        const newLinks = [];

        const filteredLinks = getLinksInSameNamespace(propNodes, propLinks);

        filteredLinks.forEach(filteredLink => {
            const link = { ...filteredLink };
            link.material = new THREE.LineBasicMaterial({
                color: 0x5a6fd9
            });
            link.geometry = new THREE.Geometry();
            link.line = new THREE.Line(link.geometry, link.material);
            newLinks.push(link);
        });

        return newLinks;
    };

    updateNodesPosition = () => {
        nodes.forEach(node => {
            const { x, y, circle } = node;
            circle.position.set(x, y, 0);
        });
    };

    updateLinksPosition = () => {
        links.forEach(link => {
            const { source, target, line } = link;
            line.geometry.verticesNeedUpdate = true;
            line.geometry.vertices[0] = new THREE.Vector3(source.x, source.y, 0);
            line.geometry.vertices[1] = new THREE.Vector3(target.x, target.y, 0);
        });
    };

    clear = () => {
        // Clear everything from the scene
        while (this.scene.children.length > 0) {
            this.scene.remove(this.scene.children[0]);
        }
        // Clear everything from the renderer
        this.renderer.renderLists.dispose();
    };

    animate = () => {
        requestAnimationFrame(this.animate);

        this.controls.update();

        this.updateNodesPosition();

        this.updateLinksPosition();

        this.renderer.render(this.scene, this.camera);
    };

    render() {
        return (
            <div
                className="network-graph flex flex-1"
                ref={ref => {
                    this.networkGraph = ref;
                }}
            />
        );
    }
}

export default NetworkGraph;
