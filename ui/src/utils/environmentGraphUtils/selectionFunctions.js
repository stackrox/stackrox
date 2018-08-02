export const enterNamespaceContainer = selection => {
    selection.classed('container', true);
    selection
        .append('rect')
        .attr('fill', 'rgba(255, 255, 255, 0.8)')
        .attr('rx', 5)
        .attr('ry', 5)
        .attr('stroke', '#9DA3C1')
        .attr('stroke-width', 1);

    selection
        .append('text')
        .attr('fill', '#3F4884')
        .style('font-size', '25px')
        .attr('dy', '1.3em')
        .attr('text-anchor', 'middle');
};

export const enterNode = callback => selection => {
    selection.classed('node cursor-pointer', true);
    selection
        .append('circle')
        .on('click', callback)
        .attr('r', 5);
};

export const updateNode = selection => {
    selection.attr('transform', d => `translate(${d.x},${d.y})`);
};

export const enterLink = selection => {
    selection
        .classed('link', true)
        .attr('stroke-width', '0.5px')
        .attr('stroke', '#3F4884')
        .attr('opacity', '0.2');
};

export const updateLink = selection => {
    selection
        .attr('x1', d => d.source.x)
        .attr('y1', d => d.source.y)
        .attr('x2', d => d.target.x)
        .attr('y2', d => d.target.y);
};

export const updateGraph = selection => {
    selection.selectAll('.node').call(updateNode);
    selection.selectAll('.link').call(updateLink);
};
