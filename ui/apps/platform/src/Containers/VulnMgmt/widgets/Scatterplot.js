import React from 'react';
import PropTypes from 'prop-types';
import { useHistory } from 'react-router-dom';
import {
    FlexibleXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    MarkSeries,
    DiscreteColorLegend,
    ChartLabel,
} from 'react-vis';
import useGraphHoverHint from 'hooks/useGraphHoverHint';

import { getHighValue, getLowValue } from './Scatterplot.utils';

const Scatterplot = ({
    data,
    lowerX,
    lowerY,
    upperX,
    upperY,
    xMultiple,
    yMultiple,
    shouldPadX,
    shouldPadY,
    plotProps,
    yAxisTitle,
    xAxisTitle,
    legendData,
}) => {
    const { onValueMouseOver, onValueMouseOut } = useGraphHoverHint();
    const history = useHistory();

    const lowX = lowerX !== null ? lowerX : getLowValue(data, 'x', xMultiple);
    const highX = upperX !== null ? upperX : getHighValue(data, 'x', xMultiple, shouldPadX);
    const xDomain = [lowX, highX];

    const lowY = lowerY !== null ? lowerY : getLowValue(data, 'y', yMultiple);
    const highY = upperY !== null ? upperY : getHighValue(data, 'y', yMultiple, shouldPadY);
    const yDomain = [lowY, highY];

    function onValueClickHandler(datum) {
        if (datum.url) {
            history.push(datum.url);
        }
    }

    return (
        <>
            <FlexibleXYPlot xDomain={xDomain} yDomain={yDomain} {...plotProps}>
                <VerticalGridLines />
                <HorizontalGridLines />
                <XAxis tickSize={0} />
                <YAxis tickSize={0} />
                <ChartLabel
                    text={xAxisTitle}
                    className="alt-x-label"
                    includeMargin={false}
                    xPercent={0.5}
                    yPercent={1.01}
                    style={{ transform: 'translate(0, 40)', textAnchor: 'middle' }}
                />
                <ChartLabel
                    text={yAxisTitle}
                    className="alt-y-label"
                    includeMargin={false}
                    xPercent={-0.01}
                    yPercent={0.5}
                    style={{ transform: 'translate(-15, 12), rotate(-90)', textAnchor: 'middle' }}
                />
                <MarkSeries
                    className="cursor-pointer"
                    colorType="literal"
                    data={data}
                    onValueMouseOver={onValueMouseOver}
                    onValueMouseOut={onValueMouseOut}
                    onValueClick={onValueClickHandler}
                />
                {legendData && (
                    <DiscreteColorLegend
                        orientation="horizontal"
                        items={legendData}
                        startTitle="CVSS SCORE"
                        style={{ position: 'absolute', bottom: '40px', right: '10px' }}
                    />
                )}
            </FlexibleXYPlot>
        </>
    );
};

Scatterplot.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            color: PropTypes.string,
            x: PropTypes.number.isRequired,
            y: PropTypes.number.isRequired,
            url: PropTypes.string,
        })
    ),
    lowerX: PropTypes.number,
    upperX: PropTypes.number,
    lowerY: PropTypes.number,
    upperY: PropTypes.number,
    xMultiple: PropTypes.number,
    yMultiple: PropTypes.number,
    shouldPadX: PropTypes.bool,
    shouldPadY: PropTypes.bool,
    plotProps: PropTypes.shape({}),
    yAxisTitle: PropTypes.string,
    xAxisTitle: PropTypes.string,
    legendData: PropTypes.arrayOf(
        PropTypes.shape({ title: PropTypes.string, color: PropTypes.string })
    ),
};

Scatterplot.defaultProps = {
    data: [],
    lowerX: null,
    upperX: null,
    lowerY: null,
    upperY: null,
    xMultiple: 10,
    yMultiple: 10,
    shouldPadX: false,
    shouldPadY: false,
    plotProps: null,
    yAxisTitle: null,
    xAxisTitle: null,
    legendData: null,
};

export default Scatterplot;
