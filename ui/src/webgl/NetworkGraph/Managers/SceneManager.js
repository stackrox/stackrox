import * as THREE from 'three';
import threeOrbitControls from 'three-orbit-controls';
import * as constants from 'constants/networkGraph';
import { intersectsNodes, intersectsNamespaces } from 'utils/networkGraphUtils';

import Node from '../sceneObjects/Node';
import ServiceLink from '../sceneObjects/ServiceLink';
import CrossNamespaceLink from '../sceneObjects/CrossNamespaceLink';
import Namespace from '../sceneObjects/Namespace';

const OrbitControls = threeOrbitControls(THREE);

const SceneManager = canvas => {
    let sceneObjects = [];
    let scene;
    let camera;
    let controls;
    let renderer;
    let raycaster;
    let mouse;
    const ZOOM_TYPE = { ZOOM_IN: true, ZOOM_OUT: false };

    let selectedNodeId;

    function buildRaycaster() {
        const webGLRayCaster = new THREE.Raycaster();
        return webGLRayCaster;
    }

    function buildMouse() {
        const webGLMouse = new THREE.Vector2();
        return webGLMouse;
    }

    function buildScene() {
        const webGLScene = new THREE.Scene();
        return webGLScene;
    }

    function buildCamera(width, height) {
        const orthographicCamera = new THREE.OrthographicCamera(
            0,
            width,
            height,
            0,
            constants.MIN_ZOOM,
            constants.MAX_ZOOM
        );
        orthographicCamera.position.z = constants.MIN_ZOOM;
        return orthographicCamera;
    }

    function buildOrbitControls() {
        const orbitControls = new OrbitControls(camera, canvas);
        Object.assign(orbitControls, constants.ORBIT_CONTROLS_CONFIG);
        return orbitControls;
    }

    function buildRenderer(width, height) {
        const config = { canvas, ...constants.RENDERER_CONFIG };
        const webGLRenderer = new THREE.WebGLRenderer(config);
        webGLRenderer.setSize(width, height);
        webGLRenderer.setPixelRatio(window.devicePixelRatio);
        return webGLRenderer;
    }

    function createSceneObjects(webGLScene, data) {
        const { nodes, links, namespaces } = data;
        const objects = [];

        namespaces.forEach(namespace => {
            objects.push(new Namespace(webGLScene, namespace));
        });

        links.forEach(link => {
            if (link.source.namespace === link.target.namespace) {
                objects.push(new ServiceLink(webGLScene, canvas, link));
            } else {
                objects.push(new CrossNamespaceLink(webGLScene, canvas, link));
            }
        });

        nodes.forEach(node => {
            objects.push(new Node(webGLScene, node));
        });

        return objects;
    }

    function clearScene() {
        // Clear everything from the scene
        while (scene.children.length > 0) {
            scene.remove(scene.children[0]);
        }
        // Clear everything from the renderer
        renderer.renderLists.dispose();
    }

    function update() {
        sceneObjects.forEach(sceneObject => {
            sceneObject.update();
        });
    }

    function render() {
        renderer.render(scene, camera);
    }

    function onCanvasResize() {
        const { clientWidth, clientHeight } = canvas;

        camera.aspect = clientWidth / clientHeight;
        camera.updateProjectionMatrix();

        renderer.setSize(clientWidth, clientHeight);
    }

    function highlightNode(deploymentId) {
        sceneObjects
            .filter(sceneObject => sceneObject.getType() === constants.NETWORK_GRAPH_TYPES.NODE)
            .forEach(node => {
                const nodeId = node.getDeploymentId();
                if ((deploymentId && nodeId === deploymentId) || !deploymentId) {
                    node.highlight();
                } else {
                    node.unhighlight();
                }
            });
    }

    function showLinks(nodeId) {
        sceneObjects
            .filter(sceneObject => sceneObject.getType() === constants.NETWORK_GRAPH_TYPES.LINK)
            .forEach(link => {
                const { deploymentId: srcDeploymentId } = link.getSource();
                const { deploymentId: tgtDeploymentId } = link.getTarget();
                const linkExists = link.isLinkInScene();
                const isLinkToNode = srcDeploymentId === nodeId || tgtDeploymentId === nodeId;

                if (!linkExists && isLinkToNode) {
                    link.createLink();
                } else if (linkExists) {
                    link.removeLink();
                    if (isLinkToNode) {
                        link.createLink();
                    }
                }
            });
    }

    function showLinksForConnectedNamespaces(obj) {
        const { namespace } = obj.object.material.userData;
        scene.children
            .filter(
                object =>
                    object.userData.namespaceLink &&
                    (object.userData.namespaceLink.source === namespace ||
                        object.userData.namespaceLink.target === namespace)
            )
            .forEach(object => {
                const { lineWidth } = object.material.uniforms;
                lineWidth.value = constants.NAMESPACE_LINK_WIDTH;
            });
    }

    function hideAllNamespaceLinks() {
        scene.children.filter(object => object.userData.namespaceLink).forEach(object => {
            const { lineWidth } = object.material.uniforms;
            lineWidth.value = 0;
        });
    }

    function getIntersectingObjects(x, y) {
        const { clientWidth, clientHeight } = canvas;
        mouse.x = (x / clientWidth) * 2 - 1;
        mouse.y = -(y / clientHeight) * 2 + 1;

        // update the ray caster with the camera and mouse position
        raycaster.setFromCamera(mouse, camera);

        // calculate objects in the scene that intersect the ray caster
        const intersects = raycaster.intersectObjects(scene.children);

        return intersects;
    }

    function getNodeAtPosition(x, y) {
        const intersectingObjects = getIntersectingObjects(x, y);

        const intersectingNodes = intersectingObjects.filter(intersectsNodes);

        if (intersectingNodes.length) {
            const { node } = intersectingNodes[0].object.material.userData;
            selectedNodeId = node.deploymentId;
            return node;
        }
        return null;
    }

    function onMouseMove(x, y) {
        const intersectingObjects = getIntersectingObjects(x, y);

        const hoveredOverNodes = intersectingObjects.filter(intersectsNodes);

        const hoveredOverNamespaces = intersectingObjects.filter(intersectsNamespaces);

        const isHovering = canvas.classList.contains('cursor-pointer');
        if (hoveredOverNodes.length) {
            if (!isHovering) {
                canvas.classList.add('cursor-pointer');
                const hoveredOverNode = hoveredOverNodes[0];
                const { deploymentId } =
                    hoveredOverNode.object.material.userData &&
                    hoveredOverNode.object.material.userData.node;
                showLinks(deploymentId);
                highlightNode(deploymentId);
                update();
            }
        } else if (isHovering) {
            canvas.classList.remove('cursor-pointer');
            showLinks(selectedNodeId);
            highlightNode(selectedNodeId);
            update();
        }

        if (hoveredOverNamespaces.length) {
            const hoveredOverNamespace = hoveredOverNamespaces[0];
            showLinksForConnectedNamespaces(hoveredOverNamespace);
        } else {
            hideAllNamespaceLinks();
        }
    }

    function calculateZoom(isZoomIn) {
        const { object, minZoom, maxZoom, update: controlsUpdate } = controls;
        const scale = 0.65 ** controls.zoomSpeed;
        if (object instanceof THREE.OrthographicCamera) {
            if (isZoomIn) {
                object.zoom = Math.max(minZoom, Math.min(maxZoom, object.zoom / scale));
            } else {
                object.zoom = Math.max(minZoom, Math.min(maxZoom, object.zoom * scale));
            }
            object.updateProjectionMatrix();
        } else {
            controls.enableZoom = false;
        }
        controlsUpdate();
    }

    function zoomIn() {
        calculateZoom(ZOOM_TYPE.ZOOM_IN);
    }

    function zoomOut() {
        calculateZoom(ZOOM_TYPE.ZOOM_OUT);
    }

    function setData(data) {
        clearScene();
        sceneObjects = createSceneObjects(scene, data);
    }

    function setUp() {
        const { clientWidth, clientHeight } = canvas;
        raycaster = buildRaycaster();
        mouse = buildMouse();
        scene = buildScene();
        camera = buildCamera(clientWidth, clientHeight);
        controls = buildOrbitControls();
        renderer = buildRenderer(clientWidth, clientHeight);
    }

    setUp();

    return {
        update,
        render,
        onMouseMove,
        getNodeAtPosition,
        onCanvasResize,
        zoomIn,
        zoomOut,
        setData
    };
};

export default SceneManager;
