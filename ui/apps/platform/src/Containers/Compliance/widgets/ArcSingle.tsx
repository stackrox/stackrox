import React, { ReactElement } from 'react';
import { XYPlot, ArcSeries, LabelSeries } from 'react-vis';

const RADIUS = 3;
const VALUE_THICKNESS = 0.2;
const FULL_ANGLE = 2 * Math.PI;

function getDataFromValue(value) {
    return [
        {
            angle0: 0,
            angle: (FULL_ANGLE * value) / 100,
            radius0: RADIUS - VALUE_THICKNESS,
            radius: RADIUS,
            color: 'var(--pf-v5-global--primary-color--100)',
        },
        {
            angle0: (FULL_ANGLE * value) / 100,
            angle: FULL_ANGLE,
            radius0: RADIUS - VALUE_THICKNESS * 0.8,
            radius: RADIUS - VALUE_THICKNESS * 0.2,
            color: 'var(--pf-v5-global--disabled-color--200)',
        },
    ];
}
const LABEL_STYLE = {
    fontSize: '36px',
    fill: 'var(--pf-v5-global--Color--100)',
    fontWeight: 500,
};

export type ArcSingleProps = {
    value: number;
};

function ArcSingle({ value }: ArcSingleProps): ReactElement {
    const data = getDataFromValue(value);

    return (
        <XYPlot xDomain={[-5, 5]} yDomain={[-5, 5]} width={135} height={121}>
            <LabelSeries
                labelAnchorX="middle"
                labelAnchorY="middle"
                data={[{ x: -1.8, y: -2.4, label: `${value}%`, style: LABEL_STYLE }]}
            />

            <ArcSeries
                animate
                center={{ x: -2, y: -2 }}
                data={data}
                colorType="literal"
                radiusDomain={[0, 1.8]}
            />
        </XYPlot>
    );
}

export default ArcSingle;
