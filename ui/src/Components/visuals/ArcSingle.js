import React from 'react';
import { XYPlot, ArcSeries, LabelSeries } from 'react-vis';
import PropTypes from 'prop-types';

const RADIUS = 3;
const VALUE_THICKNESS = 0.1;
const FULL_ANGLE = 2 * Math.PI;

function getDataFromValue(value) {
    return [
        {
            angle0: 0,
            angle: (FULL_ANGLE * value) / 100,
            radius0: RADIUS - VALUE_THICKNESS,
            radius: RADIUS,
            color: 'var(--primary-400)'
        },
        {
            angle0: (FULL_ANGLE * value) / 100,
            angle: FULL_ANGLE,
            radius0: RADIUS - VALUE_THICKNESS * 0.8,
            radius: RADIUS - VALUE_THICKNESS * 0.2,
            color: 'var(--base-400)'
        }
    ];
}
const LABEL_STYLE = {
    fontSize: '36px',
    fill: 'var(--primary-800)',
    fontWeight: 800
};

const ArcSingle = ({ value }) => {
    const data = getDataFromValue(value);

    return (
        <XYPlot xDomain={[-5, 5]} yDomain={[-5, 5]} width={150} height={125}>
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
                radiusDomain={[0, 2]}
            />
        </XYPlot>
    );
};

ArcSingle.propTypes = {
    value: PropTypes.number.isRequired
};

export default ArcSingle;
