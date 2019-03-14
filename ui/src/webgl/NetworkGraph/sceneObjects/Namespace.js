import * as THREE from 'three';
import * as constants from 'constants/networkGraph';
import {
    getBorderCanvas,
    getPlaneCanvas,
    getIconCanvas,
    CreateTextLabelMesh
} from 'utils/networkGraphUtils';

function getNamespaceContainerDimensions(nodes) {
    let minX = nodes[0].x;
    let minY = nodes[0].y;
    let maxX = nodes[0].x;
    let maxY = nodes[0].y;
    nodes.forEach(node => {
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
}

const Namespace = (scene, data) => {
    const namespace = data;

    let border = null;
    let borderTexture = null;
    let borderMaterial = null;
    let borderGeometry = null;

    let plane = null;
    let planeMaterial = null;
    let planeGeometry = null;
    let planeTexture = null;
    let canvasTexture = null;

    let icon = null;
    let iconMaterial = null;
    let iconGeometry = null;
    let iconTexture = null;

    let label = null;

    if (namespace.internetAccess) {
        const borderCanvas = getBorderCanvas(namespace);
        // adds texture to the border for the namespace
        borderTexture = new THREE.CanvasTexture(borderCanvas);
        borderTexture.needsUpdate = true;
        borderTexture.magFilter = THREE.NearestFilter;
        borderTexture.minFilter = THREE.LinearMipMapLinearFilter;
    }
    borderGeometry = new THREE.PlaneGeometry(1, 1);
    borderMaterial = new THREE.MeshBasicMaterial({
        map: borderTexture,
        side: THREE.DoubleSide,
        userData: {
            type: constants.NETWORK_GRAPH_TYPES.NAMESPACE,
            namespace: namespace.namespace
        }
    });
    border = new THREE.Mesh(borderGeometry, borderMaterial);

    canvasTexture = getPlaneCanvas(constants.CANVAS_BG_COLOR);
    planeTexture = new THREE.CanvasTexture(canvasTexture);
    planeGeometry = new THREE.PlaneGeometry(1, 1);
    planeMaterial = new THREE.MeshBasicMaterial({
        color: 0xffffff,
        side: THREE.DoubleSide,
        map: planeTexture
    });
    plane = new THREE.Mesh(planeGeometry, planeMaterial);

    // creates texts shown below the namespace
    label = CreateTextLabelMesh(
        namespace.namespace,
        constants.NAMESPACE_LABEL_CANVAS_SIZE,
        constants.NAMESPACE_LABEL_FONT_SIZE,
        true
    );

    scene.add(border);
    scene.add(plane);
    scene.add(label);

    if (namespace.internetAccess) {
        const iconCanvas = getIconCanvas();
        iconTexture = new THREE.Texture(iconCanvas);
        iconTexture.needsUpdate = true;
        iconGeometry = new THREE.PlaneBufferGeometry(1, 1);
        iconMaterial = new THREE.MeshBasicMaterial({
            transparent: true,
            map: iconTexture,
            side: THREE.DoubleSide
        });
        icon = new THREE.Mesh(iconGeometry, iconMaterial);
        scene.add(icon);
    }

    function update() {
        const { nodes } = namespace;
        const { x, y, width, height } = getNamespaceContainerDimensions(nodes);
        border.geometry.dispose();
        border.geometry = new THREE.PlaneGeometry(
            Math.abs(width + constants.CLUSTER_BORDER_PADDING),
            Math.abs(height - constants.CLUSTER_BORDER_PADDING)
        );
        border.position.set(x, y, 0);
        plane.geometry.dispose();
        plane.geometry = new THREE.PlaneGeometry(
            Math.abs(width + constants.CLUSTER_INNER_PADDING),
            Math.abs(height - constants.CLUSTER_INNER_PADDING)
        );
        plane.position.set(x, y, 0);
        label.position.set(
            x,
            y + (height - constants.CLUSTER_INNER_PADDING - constants.NAMESPACE_LABEL_OFFSET) / 2,
            0
        );
        // if the namespace has internet access then update the icon position
        if (namespace.internetAccess) {
            icon.geometry.dispose();
            icon.geometry = new THREE.PlaneGeometry(
                constants.INTERNET_ACCESS_ICON_WIDTH,
                constants.INTERNET_ACCESS_ICON_HEIGHT
            );
            icon.position.set(
                x + width / 2 + constants.INTERNET_ACCESS_ICON_X_OFFSET,
                y - height / 2 + constants.INTERNET_ACCESS_ICON_Y_OFFSET,
                0
            );
        }
    }

    function getType() {
        return constants.NETWORK_GRAPH_TYPES.Namespace;
    }

    function cleanUp() {
        scene.remove(border);
        scene.remove(plane);
        scene.remove(icon);
        if (borderTexture) borderTexture.dispose();
        borderMaterial.dispose();
        borderGeometry.dispose();
        planeMaterial.dispose();
        planeGeometry.dispose();
        planeTexture.dispose();
        if (iconMaterial) iconMaterial.dispose();
        if (iconGeometry) iconGeometry.dispose();
        if (iconTexture) iconTexture.dispose();
    }

    return {
        update,
        getType,
        cleanUp
    };
};

export default Namespace;
