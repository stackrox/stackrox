import React from 'react';
import PropTypes from 'prop-types';
import max from 'lodash/max';
import { withRouter } from 'react-router-dom';

import {
    FlexibleXYPlot,
    XAxis,
    VerticalGridLines,
    HorizontalBarSeries,
    LabelSeries,
    GradientDefs,
    ChartLabel,
} from 'react-vis';
import BarGradient from 'Components/visuals/BarGradient';
import useGraphHoverHint from 'hooks/useGraphHoverHint';

const NUM_TICKS = 3;

function getFormattedData(data) {
    const { length } = data;
    return data.map(({ y, ...rest }, index) => ({
        y: `${length - index}. ${y}`,
        ...rest,
    }));
}

function getLabelData(data) {
    return data.map(({ y, url }) => ({
        x: 0,
        y,
        label: y,
        url,
        yOffset: -13,
        xOffset: 10,
        style: { fill: 'var(--primary-800)', cursor: 'pointer' },
    }));
}

const LabeledBarGraph = ({ data, title, history }) => {
    const { onValueMouseOver, onValueMouseOut } = useGraphHoverHint();

    const upperBoundX = max([...data.map((datum) => datum.x)]);
    const formattedData = getFormattedData(data);
    const labelData = getLabelData(formattedData);

    function onValueClickHandler(datum) {
        if (datum.url) {
            history.push(datum.url);
        }
    }

    return (
        <>
            <FlexibleXYPlot margin={{ left: 5 }} xDomain={[0, upperBoundX]} yType="ordinal">
                <VerticalGridLines tickTotal={NUM_TICKS} />
                <GradientDefs>
                    <BarGradient />
                </GradientDefs>
                <HorizontalBarSeries
                    colorType="literal"
                    barWidth={0.2}
                    style={{
                        height: 6,
                        rx: '2px',
                        cursor: 'pointer',
                    }}
                    color="url(#horizontalGradient)"
                    data={formattedData}
                    onValueMouseOver={onValueMouseOver}
                    onValueMouseOut={onValueMouseOut}
                    onValueClick={onValueClickHandler}
                />
                <XAxis tickTotal={NUM_TICKS} />
                <ChartLabel
                    text={title}
                    className="alt-x-label"
                    includeMargin={false}
                    xPercent={0.5}
                    yPercent={1.01}
                    style={{ transform: 'translate(0, 40)', textAnchor: 'middle' }}
                />
                <LabelSeries
                    className="text-xs text-base-600"
                    labelAnchorY="text-top"
                    data={labelData}
                    onValueMouseOver={onValueMouseOver}
                    onValueMouseOut={onValueMouseOut}
                    onValueClick={onValueClickHandler}
                />
            </FlexibleXYPlot>
        </>
    );
};

const HOCLabeledBarGraph = withRouter(LabeledBarGraph);

HOCLabeledBarGraph.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            color: PropTypes.string,
            x: PropTypes.number.isRequired,
            y: PropTypes.string.isRequired,
            url: PropTypes.string,
        })
    ),
    title: PropTypes.string,
};

HOCLabeledBarGraph.defaultProps = {
    data: [],
    title: null,
};

export default HOCLabeledBarGraph;
