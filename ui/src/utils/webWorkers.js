export const networkGraphWW = `
importScripts("https://d3js.org/d3-collection.v1.min.js");
importScripts("https://d3js.org/d3-dispatch.v1.min.js");
importScripts("https://d3js.org/d3-quadtree.v1.min.js");
importScripts("https://d3js.org/d3-timer.v1.min.js");
importScripts("https://d3js.org/d3-force.v1.min.js");

self.onmessage = function(event) {
    const { nodes, links, clientHeight, clientWidth, constants, namespaces } = event.data;

    function forceCollide(alpha) {
        const quadtree = d3.quadtree()
            .x(d => d.x)
            .y(d => d.y)
            .addAll(nodes);

        nodes.forEach(d => {
            const r =
                d.radius +
                constants.MAX_RADIUS +
                Math.max(constants.PADDING, constants.CLUSTER_PADDING);
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
                        d.radius +
                        quad.data.radius +
                        (d.namespace === quad.data.namespace
                            ? constants.PADDING
                            : constants.CLUSTER_PADDING);
                    if (l < radius) {
                        l = ((l - radius) / l) * alpha;
                        d.x -= x *= l;
                        d.y -= y *= l;
                        quad.data.x += x;
                        quad.data.y += y;
                    }
                }
                return x1 > nx2 || x2 < nx1 || y1 > ny2 || y2 < ny1;
            });
        });
    }

    function forceCluster() {
        let nodes;
        let strength = 0.5;
        const f = alpha => {
            // scale + curve alpha value
            alpha *= strength * alpha;
            const centroids = nodes.filter(n => n.centroid);
            const map = {};
            centroids.forEach(centroid => {
                map[centroid.namespace] = centroid;
            });
            nodes.forEach(d => {
                const c = map[d.namespace];
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
        };
        f.initialize = _ => {
            nodes = _;
        };
        f.strength = _ => {
            strength = _ == null ? strength : _;
            return f;
        };
        return f;
    }

    const forceSimulation = d3
        .forceSimulation()
        .nodes(nodes, d => d.deploymentId)
        .force(
            'link',
            d3
                .forceLink(links)
                .id(d => d.deploymentId)
                .strength(0)
        )
        .force('charge', d3.forceManyBody())
        .force('center', d3.forceCenter(clientWidth / 2, clientHeight / 2))
        .force('collide', forceCollide(0.9))
        .force('cluster', forceCluster().strength(0.9))
        .alpha(1)
        .stop();

    // create static force layout by calculating ticks beforehand
    let i = 0;
    const n = nodes.length;
    while (i < n) {
        forceSimulation.tick();
        i += 1;
    }

    self.postMessage({ type: 'end', nodes, links, namespaces });
};`;

export const getBlobURL = response => {
    window.URL = window.URL || window.webkitURL;
    let blob;
    try {
        blob = new Blob([response], { type: 'application/javascript' });
    } catch (e) {
        // Backwards-compatibility
        window.BlobBuilder =
            window.BlobBuilder || window.WebKitBlobBuilder || window.MozBlobBuilder;
        blob = new window.BlobBuilder();
        blob.append(response);
        blob = blob.getBlob();
    }
    return URL.createObjectURL(blob);
};

export default { networkGraphWW, getBlobURL };
