import * as THREE from 'three';
import { MeshLine, MeshLineMaterial } from 'three.meshline';
import * as constants from 'constants/networkGraph';
import { selectClosestSides } from 'utils/networkGraphUtils';

const CrossNamespaceLink = (scene, canvas, data) => {
    const link = data;

    let sourceToSourceConnectorLink = null;
    let targetToTargetConnectorLink = null;
    let sourceConnectorToTargetConnectorLink = null;
    let sourceLinkEndpoint = null;
    let targetLinkEndpoint = null;
    let sourceLinkEndpointBorder = null;
    let targetLinkEndpointBorder = null;

    function getLineGeometry() {
        const geometry = new THREE.Geometry();
        geometry.vertices[0] = new THREE.Vector3(0, 0, 0);
        geometry.vertices[1] = new THREE.Vector3(0, 0, 0);
        geometry.verticesNeedUpdate = true;

        return geometry;
    }

    function getMeshLine(geometry) {
        /*
         * Using MeshLine instead of Line because due to limitations of the OpenGL Core Profile with
         * the WebGL renderer on most platforms linewidth will always be 1 regardless of the set value.
         */
        const meshLine = new MeshLine();
        meshLine.setGeometry(geometry);

        return meshLine;
    }

    function getLineMaterial(colorHexValue) {
        const materialConfig = {
            useMap: false,
            color: new THREE.Color(colorHexValue),
            opacity: 1,
            transparent: true,
            resolution: new THREE.Vector2(canvas.clientWidth, canvas.clientHeight),
            sizeAttenuation: true,
            lineWidth: constants.NODE_LINK_WIDTH
        };

        if (!link.isActive) {
            materialConfig.dashArray = constants.NODE_DASH_ARRAY;
            materialConfig.dashOffset = constants.NODE_DASH_OFFSET;
            materialConfig.dashRatio = constants.NODE_DASH_RATIO;
        }
        const material = new MeshLineMaterial(materialConfig);
        return material;
    }

    function createLine(colorHexValue) {
        const lineGeometry = getLineGeometry();
        const meshLine = getMeshLine(lineGeometry);
        const lineMaterial = getLineMaterial(colorHexValue);

        const line = new THREE.Mesh(meshLine.geometry, lineMaterial);
        line.frustumCulled = false;
        line.mLine = meshLine;
        line.geo = lineGeometry;
        line.userData = { link };

        return {
            line,
            lineGeometry,
            meshLine,
            lineMaterial
        };
    }

    function createLink() {
        sourceToSourceConnectorLink = createLine(constants.LINK_COLOR);
        targetToTargetConnectorLink = createLine(constants.LINK_COLOR);
        sourceConnectorToTargetConnectorLink = createLine(constants.NAMESPACE_LINK_COLOR);

        scene.add(sourceToSourceConnectorLink.line);
        scene.add(targetToTargetConnectorLink.line);
        scene.add(sourceConnectorToTargetConnectorLink.line);

        // create a link end connector border mesh
        const sourceLinkEndpointBorderGeometry = new THREE.CircleBufferGeometry(3, 32);
        const sourceLinkEndpointBorderMaterial = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_LINK_COLOR,
            transparent: true
        });
        sourceLinkEndpointBorder = {
            border: new THREE.Mesh(
                sourceLinkEndpointBorderGeometry,
                sourceLinkEndpointBorderMaterial
            ),
            geometry: sourceLinkEndpointBorderGeometry,
            material: sourceLinkEndpointBorderMaterial
        };

        const targetLinkEndpointBorderGeometry = new THREE.CircleBufferGeometry(3, 32);
        const targetLinkEndpointBorderMaterial = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_LINK_COLOR,
            transparent: true
        });
        targetLinkEndpointBorder = {
            border: new THREE.Mesh(
                targetLinkEndpointBorderGeometry,
                targetLinkEndpointBorderMaterial
            ),
            geometry: targetLinkEndpointBorderGeometry,
            material: targetLinkEndpointBorderMaterial
        };

        scene.add(sourceLinkEndpointBorder.border);
        scene.add(targetLinkEndpointBorder.border);

        // create a link end connector mesh
        const sourceLinkEndpointGeometry = new THREE.CircleBufferGeometry(2, 32);
        const sourceLinkEndpointMaterial = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_CONNECTION_POINT_COLOR,
            transparent: true
        });
        sourceLinkEndpoint = {
            endpoint: new THREE.Mesh(sourceLinkEndpointGeometry, sourceLinkEndpointMaterial),
            geometry: sourceLinkEndpointGeometry,
            material: sourceLinkEndpointMaterial
        };

        const targetLinkEndpointGeometry = new THREE.CircleBufferGeometry(2, 32);
        const targetLinkEndpointMaterial = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_CONNECTION_POINT_COLOR,
            transparent: true
        });
        targetLinkEndpoint = {
            endpoint: new THREE.Mesh(targetLinkEndpointGeometry, targetLinkEndpointMaterial),
            geometry: targetLinkEndpointGeometry,
            material: targetLinkEndpointMaterial
        };

        scene.add(sourceLinkEndpoint.endpoint);
        scene.add(targetLinkEndpoint.endpoint);
    }

    function disposeLink(_link) {
        _link.line.geometry.dispose();
        _link.line.material.dispose();
        _link.lineGeometry.dispose();
        _link.lineMaterial.dispose();
    }

    function disposeObject(object) {
        object.geometry.dispose();
        object.material.dispose();
    }

    function removeLink() {
        if (
            !sourceToSourceConnectorLink &&
            !targetToTargetConnectorLink &&
            !sourceConnectorToTargetConnectorLink
        )
            return;
        scene.remove(sourceToSourceConnectorLink.line);
        scene.remove(targetToTargetConnectorLink.line);
        scene.remove(sourceConnectorToTargetConnectorLink.line);
        scene.remove(sourceLinkEndpoint.endpoint);
        scene.remove(targetLinkEndpoint.endpoint);
        scene.remove(sourceLinkEndpointBorder.border);
        scene.remove(targetLinkEndpointBorder.border);
        disposeLink(sourceToSourceConnectorLink);
        disposeLink(targetToTargetConnectorLink);
        disposeLink(sourceConnectorToTargetConnectorLink);
        disposeObject(sourceLinkEndpoint);
        disposeObject(targetLinkEndpoint);
        disposeObject(sourceLinkEndpointBorder);
        disposeObject(targetLinkEndpointBorder);
        sourceToSourceConnectorLink = null;
        targetToTargetConnectorLink = null;
        sourceConnectorToTargetConnectorLink = null;
        sourceLinkEndpoint = null;
        targetLinkEndpoint = null;
        sourceLinkEndpointBorder = null;
        targetLinkEndpointBorder = null;
    }

    function getSource() {
        return link.source;
    }

    function getTarget() {
        return link.target;
    }

    function isLinkInScene() {
        if (
            !sourceToSourceConnectorLink ||
            !targetToTargetConnectorLink ||
            !sourceConnectorToTargetConnectorLink
        )
            return false;
        return (
            !!sourceToSourceConnectorLink.line &&
            !!targetToTargetConnectorLink.line &&
            !!sourceConnectorToTargetConnectorLink.line
        );
    }

    function update() {
        if (isLinkInScene()) {
            const namespacePositionMapping = {};

            const namespaceObjects = scene.children.filter(
                object => object.material.userData.type === constants.NETWORK_GRAPH_TYPES.NAMESPACE
            );

            namespaceObjects.forEach(object => {
                const { namespace } = object.material.userData;
                const { position } = object;
                const { width, height } = object.geometry.parameters;
                namespacePositionMapping[namespace] = {
                    x: position.x,
                    y: position.y,
                    width,
                    height
                };
            });

            const { source, target } = link;

            const {
                x: sourceX,
                y: sourceY,
                width: sourceWidth,
                height: sourceHeight
            } = namespacePositionMapping[source.namespace];
            const {
                x: targetX,
                y: targetY,
                width: targetWidth,
                height: targetHeight
            } = namespacePositionMapping[target.namespace];
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

            if (source && sourceSide && sourceToSourceConnectorLink.line.geo.vertices.length) {
                sourceToSourceConnectorLink.line.geo.vertices[0].x = source.x;
                sourceToSourceConnectorLink.line.geo.vertices[0].y = source.y;

                sourceToSourceConnectorLink.line.geo.vertices[1].x = sourceSide.x;
                sourceToSourceConnectorLink.line.geo.vertices[1].y = sourceSide.y;

                sourceToSourceConnectorLink.line.mLine.setGeometry(
                    sourceToSourceConnectorLink.line.geo
                );
            }

            if (target && targetSide && targetToTargetConnectorLink.line.geo) {
                targetToTargetConnectorLink.line.geo.vertices[0].x = target.x;
                targetToTargetConnectorLink.line.geo.vertices[0].y = target.y;
                targetToTargetConnectorLink.line.geo.vertices[1].x = targetSide.x;
                targetToTargetConnectorLink.line.geo.vertices[1].y = targetSide.y;
                targetToTargetConnectorLink.line.mLine.setGeometry(
                    targetToTargetConnectorLink.line.geo
                );
            }

            if (sourceSide && targetSide && sourceConnectorToTargetConnectorLink.line.geo) {
                sourceConnectorToTargetConnectorLink.line.geo.vertices[0].x = sourceSide.x;
                sourceConnectorToTargetConnectorLink.line.geo.vertices[0].y = sourceSide.y;
                sourceConnectorToTargetConnectorLink.line.geo.vertices[1].x = targetSide.x;
                sourceConnectorToTargetConnectorLink.line.geo.vertices[1].y = targetSide.y;
                sourceConnectorToTargetConnectorLink.line.mLine.setGeometry(
                    sourceConnectorToTargetConnectorLink.line.geo
                );
            }

            if (sourceSide && targetSide) {
                sourceLinkEndpointBorder.border.position.set(sourceSide.x, sourceSide.y, 0);
                targetLinkEndpointBorder.border.position.set(targetSide.x, targetSide.y, 0);
                sourceLinkEndpoint.endpoint.position.set(sourceSide.x, sourceSide.y, 0);
                targetLinkEndpoint.endpoint.position.set(targetSide.x, targetSide.y, 0);
            }
        }
    }

    function getType() {
        return constants.NETWORK_GRAPH_TYPES.LINK;
    }

    function cleanUp() {
        removeLink();
    }

    return {
        update,
        removeLink,
        createLink,
        getSource,
        getTarget,
        isLinkInScene,
        getType,
        cleanUp
    };
};

export default CrossNamespaceLink;
