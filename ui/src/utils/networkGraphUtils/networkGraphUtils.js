import { quadtree as d3QuadTree } from 'd3';
import * as constants from 'utils/networkGraphUtils/networkGraphConstants';

export const forceCluster = () => {
    let nodes;
    let strength = 0.5;
    const f = alpha => {
        // scale + curve alpha value
        /* eslint-disable */
        alpha *= strength * alpha;
        const centroids = nodes.filter(n => n.centroid);
        nodes.forEach(d => {
            const c = centroids.find(n => n.namespace === d.namespace);
            if (c === d) return;

            let x = d.x - c.x;
            let y = d.y - c.y;
            let l = Math.sqrt(x * x + y * y);
            const r = d.radius + c.radius;
            if (l !== r) {
                l = (l - r) / l * alpha;
                d.x -= x *= l;
                d.y -= y *= l;
                c.x += x;
                c.y += y;
            }
        });
        /* eslint-enable */
    };
    f.initialize = _ => {
        nodes = _;
    };
    f.strength = _ => {
        strength = _ == null ? strength : _;
        return f;
    };
    return f;
};

export const forceCollision = nodes => alpha => {
    const quadtree = d3QuadTree()
        .x(d => d.x)
        .y(d => d.y)
        .addAll(nodes);

    nodes.forEach(d => {
        const r =
            d.r + constants.MAX_RADIUS + Math.max(constants.PADDING, constants.CLUSTER_PADDING);
        const nx1 = d.x - r;
        const nx2 = d.x + r;
        const ny1 = d.y - r;
        const ny2 = d.y + r;
        quadtree.visit((quad, x1, y1, x2, y2) => {
            if (quad.data && quad.data !== d) {
                let x = d.x - quad.data.x;
                let y = d.y - quad.data.y;
                let l = Math.sqrt(x * x + y * y);
                const radius =
                    d.r +
                    quad.data.r +
                    (d.namespace === quad.data.namespace
                        ? constants.PADDING
                        : constants.CLUSTER_PADDING);
                if (l < radius) {
                    l = ((l - radius) / l) * alpha;
                    /* eslint-disable */
                    d.x -= x *= l;
                    d.y -= y *= l;
                    quad.data.x += x;
                    quad.data.y += y;
                    /* eslint-enable */
                }
            }
            return x1 > nx2 || x2 < nx1 || y1 > ny2 || y2 < ny1;
        });
    });
};

/**
 * Iterates through a list of links that contain a source and target,
 * and returns a new list of links where an link has a property "bidirectional" set to true if
 * there is an link that has the same source and targets, but is flipped the other way around
 *
 * @param {!Object[]} links list of links that contain a "source" and "target"
 * @returns {!Object[]}
 */
export const getBidirectionalLinks = links => {
    const sourceTargetToLinkMapping = {};

    links.forEach(link => {
        if (!sourceTargetToLinkMapping[`${link.source}-${link.target}`]) {
            if (!sourceTargetToLinkMapping[`${link.target}-${link.source}`]) {
                sourceTargetToLinkMapping[`${link.source}-${link.target}`] = link;
            } else {
                sourceTargetToLinkMapping[`${link.target}-${link.source}`].bidirectional = true;
            }
        }
    });

    return Object.values(sourceTargetToLinkMapping);
};

/**
 * Iterates through a list of links and returns only links in the same namespace
 *
 * @param {!Object[]} nodes list of nodes
 * @param {!Object[]} links list of links that contain a "source" and "target"
 * @returns {!Object[]}
 */
export const getLinksInSameNamespace = (nodes, links) => {
    const nodeIdToNodeMapping = {};

    nodes.forEach(d => {
        nodeIdToNodeMapping[d.id] = d;
    });

    const filteredLinks = links.filter(link => {
        const sourceNamespace = nodeIdToNodeMapping[link.source].namespace;
        const targetNamespace = nodeIdToNodeMapping[link.target].namespace;
        return sourceNamespace === targetNamespace;
    });

    return filteredLinks;
};

/**
 *  A function to filter a list of intersections through ray casting to show only nodes
 *
 * @returns {Function}
 */

export const intersectsNodes = obj => obj.object.material.userData.type === 'NODE';

/**
 *  A function to filter a list of intersections through ray casting to show only namespaces
 *
 * @returns {Function}
 */

export const intersectsNamespaces = obj =>
    obj.object.material.userData.type === constants.NETWORK_GRAPH_TYPES.NAMESPACE;

/**
 *  Function returns a canvas with some text drawn onto it
 *
 * @param {String} text text to draw on the canvas
 * @returns {!Object[]}
 */
export const getTextTexture = (text, size) => {
    const canvas = document.createElement('canvas');
    canvas.width = size * 4;
    canvas.height = size * 4;
    const ctx = canvas.getContext('2d');
    ctx.font = `${size / 3}px Open Sans`;
    ctx.fillStyle = 'transparent';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = constants.TEXT_COLOR;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(text, canvas.width / 2, canvas.height / 2);
    return canvas;
};

/**
 * Picks the Rectangular sides closest between two bounding boxes and returns the
 * xy-coordinates of both sides
 *
 * @param {Number} sourceX the source x position
 * @param {Number} sourceY the source y position
 * @param {Number} sourceWidth the source bounding box's width
 * @param {Number} sourceHeight the source bounding box's height
 * @param {Number} targetX the target x position
 * @param {Number} targetY the target y position
 * @param {Number} targetWidth the target bounding box's width
 * @param {Number} targetHeight the target bounding box's height
 * @returns {Object}
 */
export const selectClosestSides = (
    sourceX,
    sourceY,
    sourceWidth,
    sourceHeight,
    targetX,
    targetY,
    targetWidth,
    targetHeight
) => {
    let minDistance = Number.MAX_VALUE;
    let selectedSourceSide = null;
    let selectedTargetSide = null;
    const sourceTop = { x: sourceX, y: sourceY - sourceHeight / 2 };
    const sourceLeft = { x: sourceX - sourceWidth / 2, y: sourceY };
    const sourceRight = { x: sourceX + sourceWidth / 2, y: sourceY };
    const sourceBottom = { x: sourceX, y: sourceY + sourceHeight / 2 };
    const targetTop = { x: targetX, y: targetY - targetHeight / 2 };
    const targetLeft = { x: targetX - targetWidth / 2, y: targetY };
    const targetRight = { x: targetX + targetWidth / 2, y: targetY };
    const targetBottom = { x: targetX, y: targetY + targetHeight / 2 };
    const sourceSides = [sourceTop, sourceLeft, sourceRight, sourceBottom];
    const targetSides = [targetTop, targetLeft, targetRight, targetBottom];
    sourceSides.forEach(({ x: sourceSideX, y: sourceSideY }) => {
        targetSides.forEach(({ x: targetSideX, y: targetSideY }) => {
            const dx = targetSideX - sourceSideX;
            const dy = targetSideY - sourceSideY;
            const dr = Math.sqrt(dx * dx + dy * dy);
            if (dr < minDistance) {
                selectedSourceSide = { x: sourceSideX, y: sourceSideY };
                selectedTargetSide = { x: targetSideX, y: targetSideY };
                minDistance = dr;
            }
        });
    });
    return {
        sourceSide: selectedSourceSide,
        targetSide: selectedTargetSide
    };
};

/**
 *  Function returns a canvas for namespace borders
 *
 * @returns {!Object[]}
 */
export const getBorderCanvas = () => {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    canvas.width = constants.NAMESPACE_BORDER_CANVAS_WIDTH;
    canvas.height = constants.NAMESPACE_BORDER_CANVAS_HEIGHT;
    ctx.fillStyle = constants.NAMESPACE_BORDER_RECT_COLOR;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.strokeStyle = constants.NAMESPACE_INTERNET_ACCESS_BORDER_COLOR;
    ctx.setLineDash(constants.NAMESPACE_BORDER_DASH_WIDTH);
    ctx.strokeRect(0, 0, canvas.width, canvas.height);
    return canvas;
};

/**
 *  Function returns a canvas for planes
 * @param {String} color to draw on the canvas
 * @returns {!Object[]}
 */
export const getPlaneCanvas = (color = 'transparent') => {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    ctx.fillStyle = color;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    return canvas;
};

/**
 *  Function returns a canvas for ingress egress icons
 * @returns {!Object[]}
 */
export const getIconCanvas = () => {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    ctx.fillStyle = constants.INGRESS_EGRESS_ICON_BG_COLOR;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.fillStyle = constants.INGRESS_EGRESS_ICON_COLOR;
    ctx.font = '200px stackrox';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    const cloudIcon = '\ue901';
    ctx.fillText(cloudIcon, canvas.width / 2, canvas.height / 2);
    return canvas;
};

/**
 *  Function returns a canvas for node
 * @param {Object} node
 * @returns {!Object[]}
 */

export const getNodeCanvas = node => {
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    ctx.fillStyle = constants.CANVAS_BG_COLOR;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    if (node.internetAccess) {
        ctx.fillStyle = constants.INTERNET_ACCESS_NODE_BORDER_COLOR;
        ctx.font = '150px stackrox';
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        const iconPotential = '\ue900';
        ctx.fillText(iconPotential, canvas.width / 2, canvas.height / 2, canvas.width);
    }

    const iconNode = '\ue902';
    ctx.fillStyle = node.internetAccess
        ? constants.INTERNET_ACCESS_NODE_COLOR
        : constants.NODE_COLOR;
    ctx.font = '140px stackrox';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(iconNode, canvas.width / 2, canvas.height / 2, canvas.width);
    return canvas;
};
