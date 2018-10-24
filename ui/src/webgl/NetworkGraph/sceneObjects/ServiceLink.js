import * as THREE from 'three';
import * as constants from 'constants/networkGraph';

const ServiceLink = (scene, data) => {
    const link = data;
    let line = null;

    function createLink() {
        const material = new THREE.LineBasicMaterial({
            color: constants.LINK_COLOR
        });
        material.transparent = true;
        const geometry = new THREE.Geometry();
        line = new THREE.Line(geometry, material);
        line.name = constants.NETWORK_GRAPH_TYPES.LINK;
        line.geometry.verticesNeedUpdate = true;
        line.geometry.vertices[0] = new THREE.Vector3(link.source.x, link.source.y);
        line.geometry.vertices[1] = new THREE.Vector3(link.target.x, link.target.y);

        line.userData = { link };

        scene.add(line);
    }

    function removeLink() {
        if (!line) return;
        const selectedLine = scene.getObjectByName(line.name);
        scene.remove(selectedLine);
        line = null;
    }

    function getSource() {
        return link.source;
    }

    function getTarget() {
        return link.target;
    }

    function isLinkInScene() {
        return !!line;
    }

    function update() {
        const { source, target } = link;
        if (line) {
            line.geometry.vertices[0].x = source.x;
            line.geometry.vertices[0].y = source.y;
            line.geometry.vertices[1].x = target.x;
            line.geometry.vertices[1].y = target.y;
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

export default ServiceLink;
