import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';

import {
    FlexibleXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    MarkSeries,
    DiscreteColorLegend
} from 'react-vis';
import useGraphHoverHint from 'hooks/useGraphHoverHint';
import HoverHint from '../HoverHint';

import { getHighValue, getLowValue } from '../visual.helpers';

const Scatterplot = ({
    data,
    lowerX,
    lowerY,
    upperX,
    upperY,
    xMultiple,
    yMultiple,
    plotProps,
    yAxisTitle,
    xAxisTitle,
    legendData,
    history
}) => {
    const { hint, onValueMouseOver, onValueMouseOut, onMouseMove } = useGraphHoverHint();

    const lowX = lowerX !== null ? lowerX : getLowValue(data, 'x', xMultiple);
    const highX = upperX !== null ? upperX : getHighValue(data, 'x', xMultiple);
    const xDomain = [lowX, highX];

    const lowY = lowerY !== null ? lowerY : getLowValue(data, 'y', yMultiple);
    const highY = upperY !== null ? upperY : getHighValue(data, 'y', yMultiple);
    const yDomain = [lowY, highY];

    function onValueClickHandler(datum) {
        if (datum.url) history.push(datum.url);
    }

    return (
        <>
            <FlexibleXYPlot
                xDomain={xDomain}
                yDomain={yDomain}
                {...plotProps}
                onMouseMove={onMouseMove}
            >
                <VerticalGridLines />
                <HorizontalGridLines />
                <MarkSeries
                    colorType="literal"
                    data={data}
                    onValueMouseOver={onValueMouseOver}
                    onValueMouseOut={onValueMouseOut}
                    onValueClick={onValueClickHandler}
                />

                <XAxis tickSize={0} title={xAxisTitle} position="middle" />
                <YAxis tickSize={0} title={yAxisTitle} position="middle" />
                <DiscreteColorLegend
                    orientation="horizontal"
                    items={legendData}
                    startTitle="CVSS SCORE"
                    style={{ position: 'absolute', bottom: '40px', right: '10px' }}
                />
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
            y: PropTypes.number.isRequired,
            url: PropTypes.string
        })
    ),
    lowerX: PropTypes.number,
    upperX: PropTypes.number,
    lowerY: PropTypes.number,
    upperY: PropTypes.number,
    xMultiple: PropTypes.number,
    yMultiple: PropTypes.number,
    plotProps: PropTypes.shape({}),
    yAxisTitle: PropTypes.string,
    xAxisTitle: PropTypes.string,
    legendData: PropTypes.arrayOf(
        PropTypes.shape({ title: PropTypes.string, color: PropTypes.string })
    ),
    history: ReactRouterPropTypes.isRequired
};

Scatterplot.defaultProps = {
    data: [],
    lowerX: null,
    upperX: null,
    lowerY: null,
    upperY: null,
    xMultiple: 10,
    yMultiple: 10,
    plotProps: null,
    yAxisTitle: null,
    xAxisTitle: null,
    legendData: null
};

export default withRouter(Scatterplot);
