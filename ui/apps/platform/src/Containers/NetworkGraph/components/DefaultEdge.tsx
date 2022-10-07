import * as React from 'react';
import { observer } from 'mobx-react';
import {
    Edge,
    EdgeConnectorArrow,
    EdgeTerminalType,
    Layer,
    Point,
    useBendpoint,
    WithRemoveConnectorProps,
    WithSourceDragProps,
    WithTargetDragProps,
} from '@patternfly/react-topology';

type EdgeProps = {
    element: Edge;
    dragging?: boolean;
} & WithSourceDragProps &
    WithTargetDragProps &
    WithRemoveConnectorProps;

interface BendpointProps {
    point: Point;
}

const Bendpoint: React.FunctionComponent<BendpointProps> = observer(({ point }) => {
    const [hover, setHover] = React.useState(false);
    const [, ref] = useBendpoint(point);
    return (
        <circle
            ref={ref}
            cx={point.x}
            cy={point.y}
            r={5}
            fill="lightblue"
            fillOpacity={hover ? 0.8 : 0}
            onMouseOver={() => setHover(true)}
            onMouseOut={() => setHover(false)}
        />
    );
});

const DefaultEdge: React.FunctionComponent<EdgeProps> = ({
    element,
    sourceDragRef,
    targetDragRef,
    dragging,
    onShowRemoveConnector,
    onHideRemoveConnector,
}) => {
    const startPoint = element.getStartPoint();
    const endPoint = element.getEndPoint();
    const bendpoints = element.getBendpoints();
    const d = `M${startPoint.x} ${startPoint.y} ${bendpoints
        .map((b: Point) => `L${b.x} ${b.y} `)
        .join('')}L${endPoint.x} ${endPoint.y}`;

    return (
        <>
            <Layer id={dragging ? 'top' : undefined}>
                <path
                    strokeWidth={1}
                    stroke={(element.getData() && element.getData().color) || 'red'}
                    d={d}
                    fill="none"
                    onMouseEnter={onShowRemoveConnector}
                    onMouseLeave={onHideRemoveConnector}
                />
                {sourceDragRef && (
                    <circle
                        ref={sourceDragRef}
                        r={8}
                        cx={startPoint.x}
                        cy={startPoint.y}
                        fillOpacity={0}
                    />
                )}
                <EdgeConnectorArrow
                    dragRef={targetDragRef}
                    edge={element}
                    terminalType={EdgeTerminalType.directional}
                />
            </Layer>
            {bendpoints && bendpoints.map((p, i) => <Bendpoint point={p} key={i.toString()} />)}
        </>
    );
};

export default observer(DefaultEdge);
