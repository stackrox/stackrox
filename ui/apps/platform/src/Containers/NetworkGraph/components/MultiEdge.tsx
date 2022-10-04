/* eslint-disable no-plusplus */
import * as React from 'react';
import { observer } from 'mobx-react';
import { Edge, vecSum, vecScale, unitNormal } from '@patternfly/react-topology';

interface MultiEdgeProps {
    element: Edge;
}

// TODO create utiles to support this
const MultiEdge: React.FunctionComponent<MultiEdgeProps> = ({ element }) => {
    let idx = 0;
    let sum = 0;
    element
        .getGraph()
        .getEdges()
        .forEach((e) => {
            if (e === element) {
                idx = sum;
                sum++;
            } else if (
                e.getSource() === element.getSource() &&
                e.getTarget() === element.getTarget()
            ) {
                sum++;
            }
        });
    let d: string;
    const startPoint = element.getStartPoint();
    const endPoint = element.getEndPoint();
    if (idx === sum - 1 && sum % 2 === 1) {
        d = `M${startPoint.x} ${startPoint.y} L${endPoint.x} ${endPoint.y}`;
    } else {
        const pm = vecSum(
            [
                startPoint.x + (endPoint.x - startPoint.x) / 2,
                startPoint.y + (endPoint.y - startPoint.y) / 2,
            ],
            vecScale(
                (idx % 2 === 1 ? 25 : -25) * Math.ceil((idx + 1) / 2),
                unitNormal([startPoint.x, startPoint.y], [endPoint.x, endPoint.y])
            )
        );
        d = `M${startPoint.x} ${startPoint.y} Q${pm[0]} ${pm[1]} ${endPoint.x} ${endPoint.y}`;
    }

    return <path strokeWidth={2} stroke="#8d8d8d" d={d} fill="none" />;
};

export default observer(MultiEdge);
