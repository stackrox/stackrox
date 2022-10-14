import * as React from 'react';
import { observer } from 'mobx-react';
import {
    useCombineRefs,
    WithDragNodeProps,
    WithSelectionProps,
    Node,
    Rect,
    Layer,
    WithDndDropProps,
    WithDndDragProps,
    useAnchor,
    RectAnchor,
} from '@patternfly/react-topology';

type GroupProps = {
    children?: React.ReactNode;
    element: Node;
    droppable?: boolean;
    hover?: boolean;
    canDrop?: boolean;
} & WithSelectionProps &
    WithDragNodeProps &
    WithDndDragProps &
    WithDndDropProps;

const DefaultGroup: React.FunctionComponent<GroupProps> = ({
    element,
    children,
    selected,
    onSelect,
    dragNodeRef,
    dndDragRef,
    dndDropRef,
    droppable,
    hover,
    canDrop,
}) => {
    // @ts-expect-error TODO: raise type error issue with patternfly/react-topology team
    useAnchor(RectAnchor);
    const boxRef = React.useRef<Rect | null>(null);
    const refs = useCombineRefs<SVGRectElement>(
        dragNodeRef as React.Ref<SVGRectElement>,
        dndDragRef,
        dndDropRef
    );

    if (!droppable || !boxRef.current) {
        // change the box only when not dragging
        boxRef.current = element.getBounds();
    }
    let fill = '#ededed';
    if (canDrop && hover) {
        fill = 'lightgreen';
    } else if (canDrop && droppable) {
        fill = 'lightblue';
    } else if (element.getData()) {
        fill = element.getData().background;
    }

    if (element.isCollapsed()) {
        const { width, height } = element.getDimensions();
        return (
            <g>
                <rect
                    ref={refs}
                    x={0}
                    y={0}
                    width={width}
                    height={height}
                    rx={5}
                    ry={5}
                    fill={fill}
                    strokeWidth={2}
                    stroke={selected ? 'blue' : '#cdcdcd'}
                />
            </g>
        );
    }

    return (
        <Layer id="groups">
            <rect
                ref={refs}
                onClick={onSelect}
                x={boxRef.current.x}
                y={boxRef.current.y}
                width={boxRef.current.width}
                height={boxRef.current.height}
                fill={fill}
                strokeWidth={2}
                stroke={selected ? 'blue' : '#cdcdcd'}
            />
            {children}
        </Layer>
    );
};

export default observer(DefaultGroup);
