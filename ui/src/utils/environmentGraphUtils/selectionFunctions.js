export const enterNamespaceContainer = selection => {
    selection.classed('container', true);
    selection
        .append('rect')
        .attr('fill', 'rgba(255, 255, 255, 1)')
        .attr('rx', 5)
        .attr('ry', 5)
        .attr('stroke', d => (d.internetAccess ? 'hsl(316, 93%, 89%)' : '#9DA3C1'))
        .attr('stroke-width', d => (d.internetAccess ? '3px' : '1px'));

    selection
        .append('text')
        .attr('fill', '#3F4884')
        .style('font-size', '25px')
        .attr('dy', '1.3em')
        .attr('text-anchor', 'middle');
};

export const enterNode = (callback, d3Graph) => selection => {
    selection.classed('cursor-pointer node', true);

    selection
        .filter(d => d.internetAccess)
        .classed('internetAccess', d => d.internetAccess)
        .append('circle')
        .attr('r', 8);

    selection
        .append('circle')
        .attr('r', 5)
        .on('click', callback)
        .on('mouseover', node => {
            d3Graph
                .selectAll('.link.service')
                .filter(d => d.source.id === node.id || d.target.id === node.id)
                .attr('stroke-width', '0.4px')
                .attr('opacity', '0.7');
            d3Graph
                .selectAll('.node text')
                .filter(d => d.id !== node.id)
                .attr('opacity', '0.3');
        })
        .on('mouseout', () => {
            d3Graph
                .selectAll('.link.service')
                .attr('stroke-width', '0.2px')
                .attr('opacity', '0.1');
            d3Graph.selectAll('.node text').attr('opacity', null);
        });

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
        .classed('link service', true)
        .attr('stroke-width', '0.2px')
        .attr('stroke', '#3F4884')
        .attr('opacity', '0.1');
};

export const updateLink = selection => {
    selection
        .attr('x1', d => d.source.x)
        .attr('y1', d => d.source.y)
        .attr('x2', d => d.target.x)
        .attr('y2', d => d.target.y);
};

export const enterNamespaceLink = d3Graph => selection => {
    selection
        .classed('link namespace', true)
        .attr('stroke-width', '5px')
        .attr('stroke', '#3F4884')
        .attr('opacity', '0.5')
        .on('mouseover', edge => {
            d3Graph
                .selectAll('.link.namespace')
                .filter(({ source, target }) => source !== edge.source || target !== edge.target)
                .attr('opacity', '0.1');
        })
        .on('mouseout', () => {
            d3Graph.selectAll('.link.namespace').attr('opacity', '0.5');
        });
};

export const updateNamespaceLink = (selection, d3Graph) => {
    selection
        .attr('x1', d => {
            const { x, width } = d3Graph
                .select(`.namespace-${d.source}`)
                .node()
                .getBBox();
            return x + width / 2;
        })
        .attr('y1', d => {
            const { y, height } = d3Graph
                .select(`.namespace-${d.source}`)
                .node()
                .getBBox();
            return y + height / 2;
        })
        .attr('x2', d => {
            const { x, width } = d3Graph
                .select(`.namespace-${d.target}`)
                .node()
                .getBBox();
            return x + width / 2;
        })
        .attr('y2', d => {
            const { y, height } = d3Graph
                .select(`.namespace-${d.target}`)
                .node()
                .getBBox();
            return y + height / 2;
        });
};

export const updateGraph = (selection, d3Graph) => {
    selection.selectAll('.node').call(updateNode);
    selection.selectAll('.link.service').call(updateLink);
    selection.selectAll('.link.namespace').call(s => {
        updateNamespaceLink(s, d3Graph);
    });
};
