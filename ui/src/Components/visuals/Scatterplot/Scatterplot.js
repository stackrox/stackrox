import React from 'react';
import PropTypes from 'prop-types';
import {
    FlexibleXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    MarkSeries
} from 'react-vis';
import useGraphHoverHint from 'hooks/useGraphHoverHint';
import HoverHint from '../HoverHint';

import { getHighValue, getLowValue } from '../visual.helpers';

const Scatterplot = ({ data, lowerX, lowerY, upperX, upperY, xMultiple, yMultiple, plotProps }) => {
    const { hint, onValueMouseOver, onValueMouseOut, onMouseMove } = useGraphHoverHint();

    const lowX = lowerX !== null ? lowerX : getLowValue(data, 'x', xMultiple);
    const highX = upperX !== null ? upperX : getHighValue(data, 'x', xMultiple);
    const xDomain = [lowX, highX];

    const lowY = lowerY !== null ? lowerY : getLowValue(data, 'y', yMultiple);
    const highY = upperY !== null ? upperY : getHighValue(data, 'y', yMultiple);
    const yDomain = [lowY, highY];

    return (
        <>
            <FlexibleXYPlot
                xDomain={xDomain}
                yDomain={yDomain}
                {...plotProps}
                onMouseMove={onMouseMove}
            >
                <MarkSeries
                    colorType="literal"
                    data={data}
                    onValueMouseOver={onValueMouseOver}
                    onValueMouseOut={onValueMouseOut}
                />
                <VerticalGridLines />
                <HorizontalGridLines />
                <XAxis tickSize={0} />
                <YAxis tickSize={0} />
            </FlexibleXYPlot>
            {hint && hint.data && (
                <HoverHint
                    top={hint.y}
                    left={hint.x}
                    title={hint.data.title}
                    body={hint.data.body}
                />
            )}
        </>
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
    yMultiple: PropTypes.number,
    plotProps: PropTypes.shape({})
};

Scatterplot.defaultProps = {
    data: [],
    lowerX: null,
    upperX: null,
    lowerY: null,
    upperY: null,
    xMultiple: 10,
    yMultiple: 10,
    plotProps: null
};

export default Scatterplot;
