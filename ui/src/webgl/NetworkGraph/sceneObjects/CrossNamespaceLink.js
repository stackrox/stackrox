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

        return line;
    }

    function createLink() {
        sourceToSourceConnectorLink = createLine(constants.LINK_COLOR);
        targetToTargetConnectorLink = createLine(constants.LINK_COLOR);
        sourceConnectorToTargetConnectorLink = createLine(constants.NAMESPACE_LINK_COLOR);

        scene.add(sourceToSourceConnectorLink);
        scene.add(targetToTargetConnectorLink);
        scene.add(sourceConnectorToTargetConnectorLink);

        // create a link end connector border mesh
        let endpointGeometry = new THREE.CircleBufferGeometry(3, 32);
        let endpointMaterial = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_LINK_COLOR,
            transparent: true
        });
        sourceLinkEndpointBorder = new THREE.Mesh(endpointGeometry, endpointMaterial);
        targetLinkEndpointBorder = new THREE.Mesh(endpointGeometry, endpointMaterial);

        scene.add(sourceLinkEndpointBorder);
        scene.add(targetLinkEndpointBorder);

        // create a link end connector mesh
        endpointGeometry = new THREE.CircleBufferGeometry(2, 32);
        endpointMaterial = new THREE.MeshBasicMaterial({
            color: constants.NAMESPACE_CONNECTION_POINT_COLOR,
            transparent: true
        });
        sourceLinkEndpoint = new THREE.Mesh(endpointGeometry, endpointMaterial);
        targetLinkEndpoint = new THREE.Mesh(endpointGeometry, endpointMaterial);

        scene.add(sourceLinkEndpoint);
        scene.add(targetLinkEndpoint);
    }

    function removeLink() {
        if (
            !sourceToSourceConnectorLink &&
            !targetToTargetConnectorLink &&
            !sourceConnectorToTargetConnectorLink
        )
            return;
        scene.remove(sourceToSourceConnectorLink);
        scene.remove(targetToTargetConnectorLink);
        scene.remove(sourceConnectorToTargetConnectorLink);
        sourceToSourceConnectorLink = null;
        targetToTargetConnectorLink = null;
        sourceConnectorToTargetConnectorLink = null;
    }

    function getSource() {
        return link.source;
    }

    function getTarget() {
        return link.target;
    }

    function isLinkInScene() {
        return (
            !!sourceToSourceConnectorLink &&
            !!targetToTargetConnectorLink &&
            !!sourceConnectorToTargetConnectorLink
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

            sourceToSourceConnectorLink.geo.vertices[0].x = source.x;
            sourceToSourceConnectorLink.geo.vertices[0].y = source.y;
            sourceToSourceConnectorLink.geo.vertices[1].x = sourceSide.x;
            sourceToSourceConnectorLink.geo.vertices[1].y = sourceSide.y;
            sourceToSourceConnectorLink.mLine.setGeometry(sourceToSourceConnectorLink.geo);

            targetToTargetConnectorLink.geo.vertices[0].x = target.x;
            targetToTargetConnectorLink.geo.vertices[0].y = target.y;
            targetToTargetConnectorLink.geo.vertices[1].x = targetSide.x;
            targetToTargetConnectorLink.geo.vertices[1].y = targetSide.y;
            targetToTargetConnectorLink.mLine.setGeometry(targetToTargetConnectorLink.geo);

            sourceConnectorToTargetConnectorLink.geo.vertices[0].x = sourceSide.x;
            sourceConnectorToTargetConnectorLink.geo.vertices[0].y = sourceSide.y;
            sourceConnectorToTargetConnectorLink.geo.vertices[1].x = targetSide.x;
            sourceConnectorToTargetConnectorLink.geo.vertices[1].y = targetSide.y;
            sourceConnectorToTargetConnectorLink.mLine.setGeometry(
                sourceConnectorToTargetConnectorLink.geo
            );

            sourceLinkEndpointBorder.position.set(sourceSide.x, sourceSide.y, 0);
            targetLinkEndpointBorder.position.set(targetSide.x, targetSide.y, 0);
            sourceLinkEndpoint.position.set(sourceSide.x, sourceSide.y, 0);
            targetLinkEndpoint.position.set(targetSide.x, targetSide.y, 0);
        }
    }

    function getType() {
        return constants.NETWORK_GRAPH_TYPES.LINK;
    }

    return {
        update,
        removeLink,
        createLink,
        getSource,
        getTarget,
        isLinkInScene,
        getType
    };
};

export default CrossNamespaceLink;
