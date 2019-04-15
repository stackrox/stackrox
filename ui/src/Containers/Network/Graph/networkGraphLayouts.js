import { TEXT_MAX_WIDTH, NODE_WIDTH, NODE_PADDING } from 'constants/networkGraph';

const nodeWidth = TEXT_MAX_WIDTH + NODE_PADDING;
const nodeHeight = NODE_WIDTH + NODE_PADDING;

// Gets dimension metadata for a parent node given # of nodes
function getParentDimensions(nodeCount) {
    const cols = Math.ceil(Math.sqrt(nodeCount));
    const rows = Math.ceil(nodeCount / cols);
    const width = cols * nodeWidth;
    const height = rows * nodeHeight;
    return {
        width,
        height,
        rows,
        cols
    };
}

// Gets positions and dimensions for all parent nodes
export function getParentPositions(nodes, padding) {
    const NSNames = nodes.filter(node => !node.data().parent).map(parent => parent.data().id);

    // Get namespace dimensions sorted by width
    const namespaces = NSNames.map(id => {
        const nodeCount = nodes.filter(node => {
            const data = node.data();
            return data.parent && !data.side && data.parent === id;
        }).length;

        return { ...getParentDimensions(nodeCount), id, nodeCount };
    }).sort((a, b) => b.cols - a.cols);

    // lay out namespaces
    let x = 0;
    let y = 0;
    return namespaces.map(NS => {
        const { id, width, height } = NS;
        const result = {
            id,
            x,
            y,
            width,
            height
        };
        x += width + padding.x;
        y += height + padding.y;
        return result;
    });
}
// Can't use this.options inside prototypal function constructor in strict mode, so using a closure instead
let edgeGridOptions = {};

export function edgeGridLayout(options) {
    const defaults = {
        parentPadding: { bottom: 0, top: 0, left: 0, right: 0 },
        position: { x: 0, y: 0 }
    };
    edgeGridOptions = Object.assign({}, defaults, options);
}

// eslint-disable-next-line func-names
edgeGridLayout.prototype.run = function() {
    const options = edgeGridOptions;
    const { parentPadding, position, eles } = options;

    const nodes = eles.nodes().not(':parent');

    const renderNodes = nodes.not('[side]');
    const sideNodes = eles.nodes('[side]');

    if (!renderNodes.length) return this;

    const { width, height, cols } = getParentDimensions(renderNodes.length);

    // Calculate cell dimensions
    const cellWidth = nodeWidth;
    const cellHeight = nodeHeight;

    // Midpoints for sidewall nodes
    const midHeight = position.y + height / 2;
    const midWidth = position.x + width / 2;

    let currentRow = 0;
    let currentCol = 0;
    function incrementCell() {
        currentCol += 1;
        if (currentCol >= cols) {
            currentCol = 0;
            currentRow += 1;
        }
    }

    function getRenderNodePos(element) {
        if (element.locked() || element.isParent()) {
            return false;
        }
        const x = currentCol * cellWidth + cellWidth / 2 + position.x;
        const y = currentRow * cellHeight + cellHeight / 2 + position.y;
        incrementCell();
        return { x, y };
    }

    function getSideNodePos(element) {
        const { side } = element.data();
        switch (side) {
            case 'top':
                return {
                    x: midWidth,
                    y: position.y - parentPadding.top
                };
            case 'bottom':
                return {
                    x: midWidth,
                    y: position.y + height + parentPadding.bottom
                };
            case 'left':
                return {
                    x: position.x - parentPadding.left,
                    y: midHeight
                };
            case 'right':
                return {
                    x: position.x + width + parentPadding.right,
                    y: midHeight
                };
            default:
                return { x: position.x, y: position.y };
        }
    }

    renderNodes.layoutPositions(this, options, getRenderNodePos);
    sideNodes.layoutPositions(this, options, getSideNodePos);
    return this; // chaining
};
