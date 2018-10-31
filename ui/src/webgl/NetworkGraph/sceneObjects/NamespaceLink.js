import * as THREE from 'three';
import { MeshLine, MeshLineMaterial } from 'three.meshline';
import * as constants from 'constants/networkGraph';
import { selectClosestSides } from 'utils/networkGraphUtils';

const NamespaceLink = (scene, canvas, data) => {
    const link = data;

    let geometry;
    let material;

    // create a link mesh
    geometry = new THREE.Geometry();
    geometry.vertices[0] = new THREE.Vector3(0, 0, 0);
    geometry.vertices[1] = new THREE.Vector3(0, 0, 0);
    /*
     * Using MeshLine instead of Line because due to limitations of the OpenGL Core Profile with
     * the WebGL renderer on most platforms linewidth will always be 1 regardless of the set value.
     */
    const meshLine = new MeshLine();
    meshLine.setGeometry(geometry);
    const materialConfig = {
        useMap: false,
        color: new THREE.Color(constants.NAMESPACE_LINK_COLOR),
        opacity: 1,
        transparent: true,
        resolution: new THREE.Vector2(canvas.clientWidth, canvas.clientHeight),
        sizeAttenuation: true,
        lineWidth: 0
    };
    if (!link.isActive) {
        materialConfig.dashArray = constants.NODE_DASH_ARRAY;
        materialConfig.dashOffset = constants.NODE_DASH_OFFSET;
        materialConfig.dashRatio = constants.NODE_DASH_RATIO;
    }
    material = new MeshLineMaterial(materialConfig);
    const line = new THREE.Mesh(meshLine.geometry, material);
    line.frustumCulled = false;
    line.mLine = meshLine;
    line.geo = geometry;

    line.userData = {
        namespaceLink: link
    };

    scene.add(line);

    // create a link end connector border mesh
    geometry = new THREE.CircleBufferGeometry(3, 32);
    material = new THREE.MeshBasicMaterial({
        color: constants.NAMESPACE_LINK_COLOR,
        transparent: true
    });
    const sourceLinkEndpointBorder = new THREE.Mesh(geometry, material);
    const targetLinkEndpointBorder = new THREE.Mesh(geometry, material);

    scene.add(sourceLinkEndpointBorder);
    scene.add(targetLinkEndpointBorder);

    // create a link end connector mesh
    geometry = new THREE.CircleBufferGeometry(2, 32);
    material = new THREE.MeshBasicMaterial({
        color: 0xffffff,
        transparent: true
    });
    const sourceLinkEndpoint = new THREE.Mesh(geometry, material);
    const targetLinkEndpoint = new THREE.Mesh(geometry, material);

    scene.add(sourceLinkEndpoint);
    scene.add(targetLinkEndpoint);

    function update() {
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
    }

    function getType() {
        return constants.NETWORK_GRAPH_TYPES.NAMESPACE_LINK;
    }

    return {
        update,
        getType
    };
};

export default NamespaceLink;
