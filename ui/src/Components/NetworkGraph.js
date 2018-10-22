import React, { Component } from 'react';
import PropTypes from 'prop-types';

import * as THREE from 'three';
import { MeshLine, MeshLineMaterial } from 'three.meshline';
import * as d3 from 'd3';
import threeOrbitControls from 'three-orbit-controls';
import {
    forceCluster,
    getLinksInSameNamespace,
    intersectsNodes,
    intersectsNamespaces,
    getTextTexture,
    getBidirectionalLinks,
    selectClosestSides,
    getBorderCanvas,
    getPlaneCanvas,
    getIconCanvas,
    getNodeCanvas
} from 'utils/networkGraphUtils/networkGraphUtils';
import * as constants from 'utils/networkGraphUtils/networkGraphConstants';
import uniqBy from 'lodash/uniqBy';

const OrbitControls = threeOrbitControls(THREE);

let nodes = [];
let links = [];
let namespaces = [];
let namespaceLinks = [];
let simulation = null;
let isZoomIn = false;
let showLinks = false;

class NetworkGraph extends Component {
    static propTypes = {
        nodes: PropTypes.arrayOf(
            PropTypes.shape({
                id: PropTypes.string.isRequired
            })
        ).isRequired,
        links: PropTypes.arrayOf(
            PropTypes.shape({
                source: PropTypes.string.isRequired,
                target: PropTypes.string.isRequired
            })
        ).isRequired,
        onNodeClick: PropTypes.func.isRequired,
        updateKey: PropTypes.number.isRequired
    };

    componentDidMount() {
        if (this.isWebGLAvailable()) {
            this.setUpScene();
        }
    }

    shouldComponentUpdate(nextProps) {
        if (
            this.isWebGLAvailable() &&
            (!simulation || nextProps.updateKey !== this.props.updateKey)
        ) {
            // Clear the canvas
            this.clear();

            // Create objects for the scene
            namespaces = this.setUpNamespaces(nextProps.nodes);
            nodes = this.setUpNodes(nextProps.nodes);
            links = this.setUpServiceLinks(nextProps.nodes, nextProps.links);
            namespaceLinks = this.setUpNamespaceLinks(nextProps.nodes, nextProps.links);

            this.setUpForceSimulation();

            this.animate();
        }
        return false;
    }

    isWebGLAvailable = () => {
        try {
            const canvas = document.createElement('canvas');
            return !!(
                window.WebGLRenderingContext &&
                (canvas.getContext('webgl') || canvas.getContext('experimental-webgl'))
            );
        } catch (e) {
            return false;
        }
    };

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

        const hoveredOverNodes = intersectingObjects.filter(intersectsNodes);

        const hoveredOverNamespaces = intersectingObjects.filter(intersectsNamespaces);

        if (hoveredOverNodes.length) {
            this.networkGraph.classList.add('cursor-pointer');
            const hoveredOverNode = hoveredOverNodes[0];
            this.showLinksForConnectedNodes(hoveredOverNode);
        } else {
            this.networkGraph.classList.remove('cursor-pointer');
            links = links.map(data => {
                const link = { ...data };
                link.line.material.opacity = constants.VISIBLE;
                return link;
            });
        }

        if (hoveredOverNamespaces.length) {
            const hoveredOverNamespace = hoveredOverNamespaces[0];
            this.showLinksForConnectedNamespaces(hoveredOverNamespace);
        } else {
            this.hideAllNamespaceLinks();
        }
    };

    getIntersectingObjects = (x, y) => {
        const { clientWidth, clientHeight } = this.renderer.domElement;
        this.mouse.x = (x / clientWidth) * 2 - 1;
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
                this.updateNamespaceLinksPosition();
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
            let map = null;
            let canvas = null;
            let texture = null;
            if (namespace.internetAccess) {
                canvas = getBorderCanvas();
                // adds texture to the border for the namespace
                texture = new THREE.CanvasTexture(canvas);
                texture.needsUpdate = true;
                texture.magFilter = THREE.NearestFilter;
                texture.minFilter = THREE.LinearMipMapLinearFilter;
                map = texture;
            }
            let material = new THREE.MeshBasicMaterial({
                map,
                color: !namespace.internetAccess && constants.NAMESPACE_BORDER_COLOR,
                side: THREE.DoubleSide,
                userData: {
                    type: constants.NETWORK_GRAPH_TYPES.NAMESPACE,
                    namespace: newNamespace.namespace
                }
            });
            // creates border for the namespace
            newNamespace.border = new THREE.Mesh(geometry, material);

            geometry = new THREE.PlaneGeometry(1, 1);
            canvas = getPlaneCanvas(constants.CANVAS_BG_COLOR);
            texture = new THREE.CanvasTexture(canvas);

            material = new THREE.MeshBasicMaterial({
                map: texture,
                side: THREE.DoubleSide,
                userData: {
                    type: constants.NETWORK_GRAPH_TYPES.NAMESPACE,
                    namespace: newNamespace.namespace
                }
            });
            newNamespace.plane = new THREE.Mesh(geometry, material);

            newNamespace = this.createTextLabelMesh(
                newNamespace,
                newNamespace.namespace,
                constants.NAMESPACE_LABEL_SIZE
            );
            if (namespace.internetAccess) {
                canvas = getIconCanvas();
                texture = new THREE.Texture(canvas);
                texture.needsUpdate = true;
                geometry = new THREE.PlaneBufferGeometry(1, 1);
                material = new THREE.MeshBasicMaterial({
                    map: texture,
                    side: THREE.DoubleSide,
                    userData: {
                        type: constants.NETWORK_GRAPH_TYPES.NAMESPACE,
                        namespace: newNamespace.namespace
                    }
                });
                newNamespace.icon = new THREE.Mesh(geometry, material);
                this.scene.add(newNamespace.icon);
            }

            this.scene.add(newNamespace.border);
            this.scene.add(newNamespace.plane);
            return newNamespace;
        });

        return newNamespaces;
    };

    setUpServiceLinks = (propNodes, propLinks) => {
        const newLinks = [];

        const filteredLinks = getLinksInSameNamespace(propNodes, propLinks);

        filteredLinks.forEach(filteredLink => {
            const link = { ...filteredLink };

            const material = new THREE.LineBasicMaterial({
                color: constants.LINK_COLOR,
                userData: {
                    type: constants.NETWORK_GRAPH_TYPES.LINK
                }
            });
            material.transparent = true;
            const geometry = new THREE.Geometry();
            link.line = new THREE.Line(geometry, material);
            link.line.geometry.verticesNeedUpdate = true;
            link.line.geometry.vertices[0] = new THREE.Vector3(link.source.x, link.source.y);
            link.line.geometry.vertices[1] = new THREE.Vector3(link.target.x, link.target.y);

            newLinks.push(link);
        });

        return newLinks;
    };

    getLinksBetweenNamespaces = (propNodes, propLinks) => {
        const nodeIdToNodeMapping = {};

        propNodes.forEach(d => {
            nodeIdToNodeMapping[d.id] = d;
        });

        let filteredNamespaceLinks = propLinks
            .filter(link => {
                const sourceNamespace = nodeIdToNodeMapping[link.source].namespace;
                const targetNamespace = nodeIdToNodeMapping[link.target].namespace;
                return sourceNamespace !== targetNamespace;
            })
            .map(link => ({
                source: nodeIdToNodeMapping[link.source].namespace,
                target: nodeIdToNodeMapping[link.target].namespace,
                id: `${nodeIdToNodeMapping[link.source].namespace}-${
                    nodeIdToNodeMapping[link.target].namespace
                }`
            }));

        filteredNamespaceLinks = uniqBy(filteredNamespaceLinks, 'id');

        return filteredNamespaceLinks;
    };

    setUpNamespaceLinks = (propNodes, propLinks) => {
        const linksBetweenNamespaces = this.getLinksBetweenNamespaces(propNodes, propLinks);

        const bidirectionalLinks = getBidirectionalLinks(linksBetweenNamespaces).map(data => {
            let link = { ...data };

            link = this.createLinkEndpointsMesh(link);
            link = this.createNamespaceLinkMesh(link);

            return link;
        });

        return bidirectionalLinks;
    };

    getNamespaceContainerDimensions = (n, namespace) => {
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

    showLinksForConnectedNamespaces = obj => {
        const { namespace } = obj.object.material.userData;
        const connectedNamespaceLinks = namespaceLinks.filter(
            link => link.source === namespace || link.target === namespace
        );
        connectedNamespaceLinks.forEach(link => {
            const { lineWidth } = link.line.material.uniforms;
            lineWidth.value = constants.NAMESPACE_LINK_WIDTH;
        });
    };

    hideAllNamespaceLinks = () => {
        namespaceLinks.forEach(link => {
            const { lineWidth } = link.line.material.uniforms;
            lineWidth.value = 0;
        });
    };

    showLinksForConnectedNodes = node => {
        const { id } = node.object.userData;
        links = links.map(data => {
            const link = { ...data };
            if (link.source.id === id || link.target.id === id) {
                link.line.material.opacity = constants.VISIBLE;
            } else {
                link.line.material.opacity = constants.TRANSPARENT;
            }
            return link;
        });
    };

    showHideLinks = () => {
        if (!showLinks && this.camera.zoom >= constants.ZOOM_LEVEL_TO_SHOW_LINKS) {
            showLinks = true;
            links.forEach(link => {
                this.scene.add(link.line);
            });
        } else if (showLinks && this.camera.zoom < constants.ZOOM_LEVEL_TO_SHOW_LINKS) {
            showLinks = false;
            const removeLinks = this.scene.children.filter(
                child => child.material.userData.type === constants.NETWORK_GRAPH_TYPES.LINK
            );
            removeLinks.forEach(link => {
                this.scene.remove(link);
            });
        }
    };

    updateNodesPosition = () => {
        nodes.forEach(node => {
            const { x, y, circle, label } = node;
            circle.position.set(x, y, 0);
            label.position.set(x, y - constants.NODE_LABEL_OFFSET, 0);
        });
    };

    updateLinksPosition = () => {
        links.forEach(link => {
            const { source, target, line } = link;
            line.geometry.vertices[0].x = source.x;
            line.geometry.vertices[0].y = source.y;
            line.geometry.vertices[1].x = target.x;
            line.geometry.vertices[1].y = target.y;
        });
    };

    updateNamespacePositions = () => {
        namespaces.forEach(namespace => {
            const { namespace: name, plane, border, label, icon } = namespace;
            const { x, y, width, height } = this.getNamespaceContainerDimensions(nodes, name);
            border.geometry = new THREE.PlaneGeometry(
                Math.abs(width + constants.CLUSTER_BORDER_PADDING),
                Math.abs(height - constants.CLUSTER_BORDER_PADDING)
            );
            border.position.set(x, y, 0);
            plane.geometry = new THREE.PlaneGeometry(
                Math.abs(width + constants.CLUSTER_INNER_PADDING),
                Math.abs(height - constants.CLUSTER_INNER_PADDING)
            );
            plane.position.set(x, y, 0);
            icon.geometry = new THREE.PlaneGeometry(
                constants.INTERNET_ACCESS_ICON_WIDTH,
                constants.INTERNET_ACCESS_ICON_HEIGHT
            );
            icon.position.set(
                x + width / 2 + constants.INTERNET_ACCESS_ICON_X_OFFSET,
                y - height / 2 + constants.INTERNET_ACCESS_ICON_Y_OFFSET,
                0
            );
            label.position.set(
                x,
                y +
                    (height - constants.CLUSTER_INNER_PADDING - constants.NAMESPACE_LABEL_OFFSET) /
                        2,
                0
            );
        });
    };

    updateNamespaceLinksPosition = () => {
        const namespacePositionMapping = {};

        namespaces.forEach(namespace => {
            const { namespace: name } = namespace;
            const { position } = namespace.plane;
            const { width, height } = namespace.border.geometry.parameters;
            namespacePositionMapping[name] = {
                x: position.x,
                y: position.y,
                width,
                height
            };
        });

        namespaceLinks.forEach(link => {
            const {
                source,
                target,
                line,
                sourceLinkEndpoint,
                targetLinkEndpoint,
                sourceLinkEndpointBorder,
                targetLinkEndpointBorder
            } = link;
            const {
                x: sourceX,
                y: sourceY,
                width: sourceWidth,
                height: sourceHeight
            } = namespacePositionMapping[source];
            const {
                x: targetX,
                y: targetY,
                width: targetWidth,
                height: targetHeight
            } = namespacePositionMapping[target];
            const { sourceSide, targetSide } = selectClosestSides(
                sourceX,
                sourceY,
                sourceWidth,
                sourceHeight,
                targetX,
                targetY,
                targetWidth,
                targetHeight
            );
            line.geo.vertices[0].x = sourceSide.x;
            line.geo.vertices[0].y = sourceSide.y;
            line.geo.vertices[1].x = targetSide.x;
            line.geo.vertices[1].y = targetSide.y;
            sourceLinkEndpointBorder.position.set(sourceSide.x, sourceSide.y, 0);
            targetLinkEndpointBorder.position.set(targetSide.x, targetSide.y, 0);
            sourceLinkEndpoint.position.set(sourceSide.x, sourceSide.y, 0);
            targetLinkEndpoint.position.set(targetSide.x, targetSide.y, 0);
            line.mLine.setGeometry(line.geo);
        });
    };

    createNodeMesh = node => {
        const newNode = { ...node };
        const { id } = newNode;
        const nodeCanvas = getNodeCanvas(newNode);
        const nodeTexture = new THREE.Texture(nodeCanvas);
        nodeTexture.needsUpdate = true;
        const geometry = new THREE.PlaneBufferGeometry(32, 16);

        const material = new THREE.MeshBasicMaterial({
            map: nodeTexture,
            userData: {
                id,
                type: 'NODE'
            }
        });
        newNode.circle = new THREE.Mesh(geometry, material);
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

    createLinkEndpointsMesh = data => {
        const link = { ...data };

        let geometry;
        let material;

        // create a link end connector border mesh
        geometry = new THREE.CircleBufferGeometry(3, 32);
        material = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_LINK_COLOR
        });
        link.sourceLinkEndpointBorder = new THREE.Mesh(geometry, material);
        link.targetLinkEndpointBorder = new THREE.Mesh(geometry, material);
        this.scene.add(link.sourceLinkEndpointBorder);
        this.scene.add(link.targetLinkEndpointBorder);

        // create a link end connector mesh
        geometry = new THREE.CircleBufferGeometry(2, 32);
        material = new THREE.MeshBasicMaterial({
            color: 0xffffff,
            transparent: true
        });
        link.sourceLinkEndpoint = new THREE.Mesh(geometry, material);
        link.targetLinkEndpoint = new THREE.Mesh(geometry, material);

        this.scene.add(link.sourceLinkEndpoint);
        this.scene.add(link.targetLinkEndpoint);

        return link;
    };

    createNamespaceLinkMesh = data => {
        const link = { ...data };

        // create a link mesh
        const geometry = new THREE.Geometry();
        geometry.vertices[0] = new THREE.Vector3(0, 0, 0);
        geometry.vertices[1] = new THREE.Vector3(0, 0, 0);
        const meshLine = new MeshLine();
        meshLine.setGeometry(geometry);
        const material = new MeshLineMaterial({
            useMap: false,
            color: new THREE.Color(constants.NAMESPACE_LINK_COLOR),
            opacity: 1,
            resolution: new THREE.Vector2(
                this.networkGraph.clientWidth,
                this.networkGraph.clientHeight
            ),
            sizeAttenuation: true,
            lineWidth: 0,
            near: this.camera.near,
            far: this.camera.far
        });
        link.line = new THREE.Mesh(meshLine.geometry, material); // this syntax could definitely be improved!
        link.line.frustumCulled = false;
        link.line.mLine = meshLine;
        link.line.geo = geometry;

        this.scene.add(link.line);

        return link;
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

        this.showHideLinks();

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
            <div className="h-full w-full">
                <div
                    className="network-graph network-grid-bg flex h-full w-full"
                    ref={ref => {
                        this.networkGraph = ref;
                    }}
                />
            </div>
        );
    }
}

export default NetworkGraph;
