import * as React from 'react';
import {
    ComponentFactory,
    withContextMenu,
    ContextMenuSeparator,
    ContextMenuItem,
    withDragNode,
    withSelection,
    ModelKind,
    DragObjectWithType,
    Node,
    withPanZoom,
    GraphComponent,
    withCreateConnector,
    Graph,
    isNode,
    withDndDrop,
    Edge,
    withTargetDrag,
    withSourceDrag,
    nodeDragSourceSpec,
    nodeDropTargetSpec,
    groupDropTargetSpec,
    graphDropTargetSpec,
    NODE_DRAG_TYPE,
    CREATE_CONNECTOR_DROP_TYPE,
} from '@patternfly/react-topology';
import StyleNode from './StyleNode';
import StyleGroup from './StyleGroup';
import StyleEdge from './StyleEdge';
import CustomPathNode from './CustomPathNode';
import CustomPolygonNode from './CustomPolygonNode';

const CONNECTOR_SOURCE_DROP = 'connector-src-drop';
const CONNECTOR_TARGET_DROP = 'connector-target-drop';

interface EdgeProps {
    element: Edge;
}

const contextMenuItem = (label: string, i: number): React.ReactElement => {
    if (label === '-') {
        return <ContextMenuSeparator key={`separator:${i.toString()}`} />;
    }
    return (
        // eslint-disable-next-line no-alert
        <ContextMenuItem key={label} onClick={() => alert(`Selected: ${label}`)}>
            {label}
        </ContextMenuItem>
    );
};

const createContextMenuItems = (...labels: string[]): React.ReactElement[] =>
    labels.map(contextMenuItem);

const defaultMenu = createContextMenuItems('First', 'Second', 'Third', '-', 'Fourth');

const stylesComponentFactory: ComponentFactory = (kind: ModelKind, type: string): any => {
    if (kind === ModelKind.graph) {
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-ignore
        return withDndDrop(graphDropTargetSpec([NODE_DRAG_TYPE]))(withPanZoom()(GraphComponent));
    }
    switch (type) {
        case 'node':
            return withCreateConnector((source: Node, target: Node | Graph): void => {
                let targetId;
                const model = source.getController().toModel();
                if (isNode(target)) {
                    targetId = target.getId();
                } else {
                    return;
                }
                const id = `e${source.getGraph().getEdges().length + 1}`;
                if (!model.edges) {
                    model.edges = [];
                }
                model.edges.push({
                    id,
                    type: 'edge',
                    source: source.getId(),
                    target: targetId,
                });
                source.getController().fromModel(model);
            })(
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore

                withDndDrop(
                    nodeDropTargetSpec([
                        CONNECTOR_SOURCE_DROP,
                        CONNECTOR_TARGET_DROP,
                        CREATE_CONNECTOR_DROP_TYPE,
                    ])
                )(
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                    withContextMenu(() => defaultMenu)(
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore

                        withDragNode(nodeDragSourceSpec('node', true, true))(
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            withSelection()(StyleNode)
                        )
                    )
                )
            );
        case 'node-path':
            return CustomPathNode;
        case 'node-polygon':
            return CustomPolygonNode;
        case 'group':
            return withDndDrop(groupDropTargetSpec)(
                // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                // @ts-ignore
                withContextMenu(() => defaultMenu)(
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                    withDragNode(nodeDragSourceSpec('group'))(withSelection()(StyleGroup))
                )
            );
        case 'edge':
            return withSourceDrag<DragObjectWithType, Node, any, EdgeProps>({
                item: { type: CONNECTOR_SOURCE_DROP },
                begin: (monitor, props) => {
                    props.element.raise();
                    return props.element;
                },
                drag: (event, monitor, props) => {
                    props.element.setStartPoint(event.x, event.y);
                },
                end: (dropResult, monitor, props) => {
                    if (monitor.didDrop() && dropResult && props) {
                        props.element.setSource(dropResult);
                    }
                    props.element.setStartPoint();
                },
            })(
                withTargetDrag<DragObjectWithType, Node, { dragging?: boolean }, EdgeProps>({
                    item: { type: CONNECTOR_TARGET_DROP },
                    begin: (monitor, props) => {
                        props.element.raise();
                        return props.element;
                    },
                    drag: (event, monitor, props) => {
                        props.element.setEndPoint(event.x, event.y);
                    },
                    end: (dropResult, monitor, props) => {
                        if (monitor.didDrop() && dropResult && props) {
                            props.element.setTarget(dropResult);
                        }
                        props.element.setEndPoint();
                    },
                    collect: (monitor) => ({
                        dragging: monitor.isDragging(),
                    }),
                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                    // @ts-ignore
                })(withContextMenu(() => defaultMenu)(withSelection()(StyleEdge)))
            );
        default:
            return undefined;
    }
};

export default stylesComponentFactory;
