/* eslint-disable @typescript-eslint/no-explicit-any */
import {
    ComponentFactory,
    withDragNode,
    withSelection,
    ModelKind,
    withPanZoom,
    GraphComponent,
    withDndDrop,
    nodeDragSourceSpec,
    graphDropTargetSpec,
    NODE_DRAG_TYPE,
} from '@patternfly/react-topology';
import StyleNode from './StyleNode';
import StyleGroup from './StyleGroup';
import StyleEdge from './StyleEdge';
import StyleFakeGroup from './StyleFakeGroup';

const stylesComponentFactory: ComponentFactory = (kind: ModelKind, type: string): any => {
    if (kind === ModelKind.graph) {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        return withDndDrop(graphDropTargetSpec([NODE_DRAG_TYPE]))(withPanZoom()(GraphComponent));
    }
    switch (type) {
        case 'node':
            return withDragNode(nodeDragSourceSpec('node', true, true))(
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                withSelection()(StyleNode)
            );
        case 'group':
            return withDragNode(nodeDragSourceSpec('group'))(
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                withSelection()(StyleGroup)
            );
        case 'fakeGroup':
            return withDragNode(nodeDragSourceSpec('node', true, true))(
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                withSelection()(StyleFakeGroup)
            );
        case 'edge':
            return StyleEdge;
        default:
            return undefined;
    }
};

export default stylesComponentFactory;
