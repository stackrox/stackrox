import React from 'react';
import PropTypes from 'prop-types';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    MarkSeries
} from 'react-vis';

import { getHighValue, getLowValue } from '../visual.helpers';

const Scatterplot = ({ data, lowerX, lowerY, upperX, upperY, xMultiple, yMultiple }) => {
    const lowX = lowerX !== null ? lowerX : getLowValue(data, 'x', xMultiple);
    const highX = upperX !== null ? upperX : getHighValue(data, 'x', xMultiple);
    const xDomain = [lowX, highX];

    const lowY = lowerY !== null ? lowerY : getLowValue(data, 'y', yMultiple);
    const highY = upperY !== null ? upperY : getHighValue(data, 'y', yMultiple);
    const yDomain = [lowY, highY];

    return (
        <FlexibleWidthXYPlot height={200} xDomain={xDomain} yDomain={yDomain}>
            <MarkSeries colorType="literal" data={data} />
            <VerticalGridLines />
            <HorizontalGridLines />
            <XAxis tickSize={0} />
            <YAxis tickSize={0} />
        </FlexibleWidthXYPlot>
    );
};

Scatterplot.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            color: PropTypes.string,
            x: PropTypes.number.isRequired,
            y: PropTypes.number.isRequired
        })
    ),
    lowerX: PropTypes.number,
    upperX: PropTypes.number,
    lowerY: PropTypes.number,
    upperY: PropTypes.number,
    xMultiple: PropTypes.number,
    yMultiple: PropTypes.number
};

Scatterplot.defaultProps = {
    data: [],
    lowerX: null,
    upperX: null,
    lowerY: null,
    upperY: null,
    xMultiple: 10,
    yMultiple: 10
};

export default Scatterplot;
