// Helper functions

/**
 * Picks the Rectangular sides closest between two bounding boxes and returns the
 * xy-coordinates of both sides and the distance between the two points
 *
 * @param {Object} source the source bounding box with x,y,width, and height values
 * @param {Object} target the target bounding box with x,y,width, and height values
 * @returns {Object}
 */
const selectClosestSides = (
    { x: sourceX, y: sourceY, width: sourceWidth, height: sourceHeight },
    { x: targetX, y: targetY, width: targetWidth, height: targetHeight }
) => {
    let minDistance = Number.MAX_VALUE;
    let selectedSourceSide = null;
    let selectedTargetSide = null;
    const sourceTop = { x: sourceX + sourceWidth / 2, y: sourceY };
    const sourceLeft = { x: sourceX, y: sourceY + sourceHeight / 2 };
    const sourceRight = { x: sourceX + sourceWidth, y: sourceLeft.y };
    const sourceBottom = { x: sourceTop.x, y: sourceY + sourceHeight };
    const targetTop = { x: targetX + targetWidth / 2, y: targetY };
    const targetLeft = { x: targetX, y: targetY + targetHeight / 2 };
    const targetRight = { x: targetX + targetWidth, y: targetLeft.y };
    const targetBottom = { x: targetTop.x, y: targetY + targetHeight };
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
        targetSide: selectedTargetSide,
        minDistance
    };
};

// Selection Functions

export const enterNamespaceContainer = d3Graph => selection => {
    selection.classed('container', true);
    selection
        .append('rect')
        .attr('class', d => `namespace-${d.namespace}`)
        .attr('fill', 'rgba(255, 255, 255, 1)')
        .attr('rx', 5)
        .attr('ry', 5)
        .attr('stroke', d => (d.internetAccess ? 'hsl(316, 93%, 89%)' : '#9DA3C1'))
        .attr('stroke-width', d => (d.internetAccess ? '3px' : '1px'))
        .on('mouseover', d => {
            d3Graph
                .selectAll('.link.namespace')
                .filter(({ source, target }) => source === d.namespace || target === d.namespace)
                .attr('opacity', '1');
        })
        .on('mouseout', () => {
            d3Graph.selectAll('.link.namespace').attr('opacity', '0.2');
        });

    selection
        .append('text')
        .attr('fill', '#3F4884')
        .style('font-size', '25px')
        .attr('dy', '1.3em')
        .attr('text-anchor', 'middle');
};

export const enterNode = (callback, d3Graph) => selection => {
    selection
        .classed('cursor-pointer node', true)
        .on('mouseover', node => {
            d3Graph
                .selectAll('.link.service')
                .filter(d => d.source.id === node.id || d.target.id === node.id)
                .attr('opacity', '1');
            d3Graph
                .selectAll('.node text')
                .filter(d => d.id !== node.id)
                .attr('opacity', '0.3');
        })
        .on('mouseout', () => {
            d3Graph.selectAll('.link.service').attr('opacity', '0.1');
            d3Graph.selectAll('.node text').attr('opacity', null);
        });

    selection
        .filter(d => d.internetAccess)
        .classed('internetAccess', d => d.internetAccess)
        .append('circle')
        .attr('r', 8);

    selection
        .append('circle')
        .attr('r', 5)
        .on('click', callback);

    selection
        .append('text')
        .attr('fill', '#3F4884')
        .style('font-size', '12px')
        .attr('dy', '2em')
        .attr('text-anchor', 'middle')
        .classed('pointer-events-none', true)
        .text(d => d.deploymentName);
};

export const updateNode = selection => {
    selection.attr('transform', d => `translate(${d.x},${d.y})`);
};

export const enterLink = selection => {
    selection
        .classed('link service pointer-events-none', true)
        .attr('fill', 'none')
        .attr('stroke', '#3F4884')
        .attr('marker-end', 'url(#end)')
        .attr('marker-start', d => {
            if (d.bidirectional) return 'url(#start)';
            return null;
        })
        .attr('opacity', '0.1');
};

export const updateLink = selection => {
    selection.attr('d', ({ source, target }) => {
        const dx = target.x - source.x;
        const dy = target.y - source.y;
        const dr = Math.sqrt(dx * dx + dy * dy);
        return `M${source.x},${source.y}A${dr},${dr} 0 0,1 ${target.x},${target.y}`;
    });
};

export const enterNamespaceLink = selection => {
    selection
        .classed('link namespace', true)
        .attr('fill', 'none')
        .attr('stroke', '#3F4884')
        .attr('marker-end', 'url(#end)')
        .attr('marker-start', d => {
            if (d.bidirectional) return 'url(#start)';
            return null;
        })
        .attr('opacity', '0.2');
};

export const updateNamespaceLink = (selection, d3Graph) => {
    selection.attr('d', d => {
        const source = d3Graph
            .select(`.namespace-${d.source}`)
            .node()
            .getBBox();
        const target = d3Graph
            .select(`.namespace-${d.target}`)
            .node()
            .getBBox();
        const { sourceSide, targetSide, minDistance } = selectClosestSides(source, target);
        return `M${sourceSide.x},${sourceSide.y}A${minDistance},${minDistance} 0 0,1 ${
            targetSide.x
        },${targetSide.y}`;
    });
};

export const updateGraph = (selection, d3Graph) => {
    selection.selectAll('.node').call(updateNode);
    selection.selectAll('.link.service').call(updateLink);
    selection.selectAll('.link.namespace').call(s => {
        updateNamespaceLink(s, d3Graph);
    });
};
