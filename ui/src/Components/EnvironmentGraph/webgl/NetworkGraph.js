import React, { Component } from 'react';
import PropTypes from 'prop-types';

import * as THREE from 'three';
import * as d3 from 'd3';
import threeOrbitControls from 'three-orbit-controls';
import {
    forceCluster,
    getLinksInSameNamespace,
    intersectsNodes,
    getTextTexture
} from 'utils/networkGraphUtils/networkGraphUtils';
import * as constants from 'utils/networkGraphUtils/networkGraphConstants';
import * as Icon from 'react-feather';

const OrbitControls = threeOrbitControls(THREE);

let nodes = [];
let links = [];
let namespaces = [];
let simulation = null;
let isZoomIn = false;

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
            namespaces = this.setUpNamespaces(nextProps.nodes);
            nodes = this.setUpNodes(nextProps.nodes);
            links = this.setUpLinks(nextProps.nodes, nextProps.links);

            this.setUpForceSimulation();

            this.animate();
        }

        return false;
    }

    onGraphClick = ({ layerX: x, layerY: y }) => {
        const intersectingObjects = this.getIntersectingObjects(x, y);

        const intersectingNodes = intersectingObjects.filter(intersectsNodes);

        if (intersectingNodes.length) {
            const node = nodes.find(
                n =>
                    n.circle &&
                    n.circle.position.x === intersectingNodes[0].object.position.x &&
                    n.circle.position.y === intersectingNodes[0].object.position.y
            );
            this.props.onNodeClick(node);
        }
    };

    onMouseMove = ({ layerX: x, layerY: y }) => {
        const intersectingObjects = this.getIntersectingObjects(x, y);

        const isHoveringOverNode = intersectingObjects.filter(intersectsNodes).length;

        if (isHoveringOverNode) {
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
                forceCluster().strength(constants.FORCE_CONFIG.FORCE_CLUSTER_STRENGTH)
            )
            .on('tick', () => {
                this.updateNamespacePositions();
                this.updateNodesPosition();
                this.updateLinksPosition();
            })
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
        const namespacesMapping = {};

        propNodes.forEach(propNode => {
            let modifiedNode;
            const node = { ...propNode };
            node.radius = 1;

            modifiedNode = this.createNodeMesh(node, namespacesMapping);

            // set centroid
            if (!namespacesMapping[modifiedNode.namespace]) {
                modifiedNode.centroid = true;
                namespacesMapping[modifiedNode.namespace] = modifiedNode;
            }
            if (modifiedNode.internetAccess) {
                namespacesMapping[modifiedNode.namespace] = modifiedNode;
            }

            modifiedNode = this.createTextLabelMesh(
                modifiedNode,
                modifiedNode.deploymentName,
                constants.NODE_LABEL_SIZE
            );

            newNodes.push(modifiedNode);
        });

        return newNodes;
    };

    setUpNamespaces = propNodes => {
        const namespacesMapping = {};
        let newNamespaces = [];

        propNodes.forEach(propNode => {
            if (!namespacesMapping[propNode.namespace] || propNode.internetAccess) {
                const namespace = {
                    namespace: propNode.namespace,
                    internetAccess: propNode.internetAccess
                };

                namespacesMapping[propNode.namespace] = namespace;
            }
        });

        newNamespaces = Object.values(namespacesMapping).map(namespace => {
            let newNamespace = { ...namespace };

            let geometry = new THREE.PlaneGeometry(1, 1);
            let material = new THREE.MeshBasicMaterial({
                color: namespace.internetAccess
                    ? constants.INTERNET_ACCESS_COLOR
                    : constants.NAMESPACE_BORDER_COLOR,
                side: THREE.DoubleSide
            });
            newNamespace.border = new THREE.Mesh(geometry, material);

            geometry = new THREE.PlaneGeometry(1, 1);
            material = new THREE.MeshBasicMaterial({
                color: 0xffffff,
                side: THREE.DoubleSide
            });
            newNamespace.plane = new THREE.Mesh(geometry, material);

            newNamespace = this.createTextLabelMesh(
                newNamespace,
                newNamespace.namespace,
                constants.NAMESPACE_LABEL_SIZE
            );

            this.scene.add(newNamespace.border);
            this.scene.add(newNamespace.plane);

            return newNamespace;
        });

        return newNamespaces;
    };

    setUpLinks = (propNodes, propLinks) => {
        const newLinks = [];

        const filteredLinks = getLinksInSameNamespace(propNodes, propLinks);

        filteredLinks.forEach(filteredLink => {
            const link = { ...filteredLink };
            link.material = new THREE.LineBasicMaterial({
                color: constants.LINK_COLOR
            });
            link.geometry = new THREE.Geometry();
            link.line = new THREE.Line(link.geometry, link.material);
            newLinks.push(link);
        });

        return newLinks;
    };

    getNamespaceDimensions = (n, namespace) => {
        const filteredNodes = n.filter(node => node.namespace === namespace);
        let minX = filteredNodes[0].x;
        let minY = filteredNodes[0].y;
        let maxX = filteredNodes[0].x;
        let maxY = filteredNodes[0].y;
        filteredNodes.forEach(node => {
            if (node.x < minX) minX = node.x;
            if (node.y > minY) minY = node.y;
            if (node.x > maxX) maxX = node.x;
            if (node.y < maxY) maxY = node.y;
        });
        const x = minX;
        const y = minY;
        const width = maxX - minX;
        const height = maxY - minY;
        return {
            x: x + width / 2,
            y: y + height / 2,
            width,
            height
        };
    };

    updateNodesPosition = () => {
        nodes.forEach(node => {
            const { x, y, circle, border, label } = node;
            border.position.set(x, y, 0);
            circle.position.set(x, y, 0);
            label.position.set(x, y - constants.NODE_LABEL_OFFSET, 0);
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

    createNodeMesh = node => {
        const newNode = { ...node };

        let geometry = new THREE.CircleBufferGeometry(8, 32);
        let material = new THREE.MeshBasicMaterial({
            color: constants.INTERNET_ACCESS_COLOR,
            transparent: !newNode.internetAccess,
            opacity: newNode.internetAccess ? 1 : 0
        });
        newNode.border = new THREE.Mesh(geometry, material);

        geometry = new THREE.CircleBufferGeometry(5, 32);
        material = new THREE.MeshBasicMaterial({
            color: constants.NODE_COLOR
        });
        newNode.circle = new THREE.Mesh(geometry, material);

        this.scene.add(newNode.border);
        this.scene.add(newNode.circle);

        return newNode;
    };

    createTextLabelMesh = (data, text, size) => {
        const modifiedData = { ...data };
        const trimmedName = text.length > 15 ? `${text.substring(0, 15)}...` : text;

        const canvasTexture = getTextTexture(trimmedName, size);

        const texture = new THREE.Texture(canvasTexture);
        texture.needsUpdate = true;
        const material = new THREE.MeshBasicMaterial({ map: texture, side: THREE.DoubleSide });
        material.transparent = true;
        const geometry = new THREE.PlaneBufferGeometry(size, size);
        modifiedData.label = new THREE.Mesh(geometry, material);

        this.scene.add(modifiedData.label);

        return modifiedData;
    };

    updateNamespacePositions = () => {
        namespaces.forEach(namespace => {
            const { namespace: name, plane, border, label } = namespace;
            const { x, y, width, height } = this.getNamespaceDimensions(nodes, name);
            border.geometry = new THREE.PlaneGeometry(
                width + constants.CLUSTER_BORDER_PADDING,
                height - constants.CLUSTER_BORDER_PADDING
            );
            border.position.set(x, y, 0);
            plane.geometry = new THREE.PlaneGeometry(
                width + constants.CLUSTER_INNER_PADDING,
                height - constants.CLUSTER_INNER_PADDING
            );
            plane.position.set(x, y, 0);
            label.position.set(
                x,
                y +
                    (height - constants.CLUSTER_INNER_PADDING - constants.NAMESPACE_LABEL_OFFSET) /
                        2,
                0
            );
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

        this.renderer.render(this.scene, this.camera);
    };

    zoomIn = () => {
        isZoomIn = true;
        this.calculateZoom();
    };

    zoomOut = () => {
        isZoomIn = false;
        this.calculateZoom();
    };

    calculateZoom = () => {
        const { object, minZoom, maxZoom, update } = this.controls;
        const scale = 0.65 ** this.controls.zoomSpeed;
        if (object instanceof THREE.OrthographicCamera) {
            if (isZoomIn) {
                object.zoom = Math.max(minZoom, Math.min(maxZoom, object.zoom / scale));
            } else {
                object.zoom = Math.max(minZoom, Math.min(maxZoom, object.zoom * scale));
            }
            object.updateProjectionMatrix();
        } else {
            this.controls.enableZoom = false;
        }
        update();
    };

    render() {
        return (
            <div className="h-full w-full relative">
                <div
                    className="network-graph flex h-full w-full"
                    ref={ref => {
                        this.networkGraph = ref;
                    }}
                />
                <div className="graph-zoom-buttons m-4 absolute pin-b pin-r z-20">
                    <button className="btn-icon btn-primary mb-2" onClick={this.zoomIn}>
                        <Icon.Plus className="h-4 w-4" />
                    </button>
                    <button className="btn-icon btn-primary" onClick={this.zoomOut}>
                        <Icon.Minus className="h-4 w-4" />
                    </button>
                </div>
            </div>
        );
    }
}

export default NetworkGraph;
