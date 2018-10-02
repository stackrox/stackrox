import { quadtree as d3QuadTree } from 'd3';
import {
    TEXT_COLOR,
    MAX_RADIUS,
    PADDING,
    CLUSTER_PADDING
} from 'utils/networkGraphUtils/networkGraphConstants';

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
        const r = d.r + MAX_RADIUS + Math.max(PADDING, CLUSTER_PADDING);
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
                    (d.namespace === quad.data.namespace ? PADDING : CLUSTER_PADDING);
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
 * Iterates through a list of edges that contain a source and target,
 * and returns a new list of edges where an edge has a property "bidirectional" set to true if
 * there is an edge that has the same source and targets, but is flipped the other way around
 *
 * @param {!Object[]} edges list of edges that contain a "source" and "target"
 * @returns {!Object[]}
 */
export const getBidirectionalEdges = edges => {
    const sourceTargetToEdgeMapping = {};

    edges.forEach(edge => {
        if (!sourceTargetToEdgeMapping[`${edge.source}-${edge.target}`]) {
            if (!sourceTargetToEdgeMapping[`${edge.target}-${edge.source}`]) {
                sourceTargetToEdgeMapping[`${edge.source}-${edge.target}`] = edge;
            } else {
                sourceTargetToEdgeMapping[`${edge.target}-${edge.source}`].bidirectional = true;
            }
        }
    });

    return Object.values(sourceTargetToEdgeMapping);
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

export const intersectsNodes = obj => obj.object.geometry.type === 'CircleBufferGeometry';

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
    ctx.fillStyle = TEXT_COLOR;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(text, canvas.width / 2, canvas.height / 2);
    return canvas;
};
