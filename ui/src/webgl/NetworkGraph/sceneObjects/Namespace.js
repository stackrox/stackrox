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
    let plane = null;
    let label = null;
    let icon = null;

    let geometry = new THREE.PlaneGeometry(1, 1);
    let canvas = null;
    let texture = null;
    if (namespace.internetAccess) {
        canvas = getBorderCanvas(namespace);
        // adds texture to the border for the namespace
        texture = new THREE.CanvasTexture(canvas);
        texture.needsUpdate = true;
        texture.magFilter = THREE.NearestFilter;
        texture.minFilter = THREE.LinearMipMapLinearFilter;
    }
    let material = new THREE.MeshBasicMaterial({
        map: texture,
        side: THREE.DoubleSide,
        userData: {
            type: constants.NETWORK_GRAPH_TYPES.NAMESPACE,
            namespace: namespace.namespace
        }
    });
    // creates border for the namespace
    border = new THREE.Mesh(geometry, material);
    canvas = getPlaneCanvas(constants.CANVAS_BG_COLOR);
    texture = new THREE.CanvasTexture(canvas);
    geometry = new THREE.PlaneGeometry(1, 1);
    material = new THREE.MeshBasicMaterial({
        color: 0xffffff,
        side: THREE.DoubleSide
    });
    plane = new THREE.Mesh(geometry, material);

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
        canvas = getIconCanvas();
        texture = new THREE.Texture(canvas);
        texture.needsUpdate = true;
        geometry = new THREE.PlaneBufferGeometry(1, 1);
        material = new THREE.MeshBasicMaterial({
            transparent: true,
            map: texture,
            side: THREE.DoubleSide
        });
        icon = new THREE.Mesh(geometry, material);
        scene.add(icon);
    }

    function update() {
        const { nodes } = namespace;
        const { x, y, width, height } = getNamespaceContainerDimensions(nodes);
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
        label.position.set(
            x,
            y + (height - constants.CLUSTER_INNER_PADDING - constants.NAMESPACE_LABEL_OFFSET) / 2,
            0
        );
        // if the namespace has internet access then update the icon position
        if (namespace.internetAccess) {
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

    return {
        update,
        getType
    };
};

export default Namespace;
