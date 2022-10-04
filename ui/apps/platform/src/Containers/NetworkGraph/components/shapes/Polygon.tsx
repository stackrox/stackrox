import { PointTuple, ShapeProps, usePolygonAnchor } from '@patternfly/react-topology';
import * as React from 'react';

const Polygon: React.FunctionComponent<ShapeProps> = ({
    className,
    width,
    height,
    filter,
    dndDropRef,
}) => {
    const points: PointTuple[] = React.useMemo(
        () => [
            [width / 2, 0],
            [width - width / 8, height],
            [0, height / 3],
            [width, height / 3],
            [width / 8, height],
        ],
        [height, width]
    );
    usePolygonAnchor(points);
    return (
        <polygon
            className={className}
            ref={dndDropRef}
            points={points.map((p) => `${p[0]},${p[1]}`).join(' ')}
            filter={filter}
        />
    );
};

export default Polygon;
