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
    GradientDefs
} from 'react-vis';
import BarGradient from 'Components/visuals/BarGradient';

function getXYPlotHeight(data) {
    return 36 * data.length; // 36px is allotted per bar to allow the bars to fit inside the XYPlot graph
}

function getFormattedData(data) {
    const { length } = data;
    return data.map(({ y, ...rest }, index) => ({
        y: `${length - index}. ${y}`,
        ...rest
    }));
}

function getLabelData(data) {
    return data.map(({ y, url }) => ({
        x: 0,
        y,
        label: y,
        url,
        yOffset: -7,
        xOffset: 10,
        style: { fill: 'var(--primary-800)', cursor: 'pointer' }
    }));
}

const LabeledBarGraph = ({ data, title, history }) => {
    const upperBoundX = max([10, ...data.map(datum => datum.x)]);
    const formattedData = getFormattedData(data);
    const labelData = getLabelData(formattedData);

    function onValueClickHandler(datum) {
        if (datum.url) history.push(datum.url);
    }

    return (
        <FlexibleXYPlot
            height={getXYPlotHeight(data)}
            margin={{ left: 5 }}
            xDomain={[0, upperBoundX]}
            yType="ordinal"
        >
            <VerticalGridLines tickTotal={upperBoundX / 2} />
            <GradientDefs>
                <BarGradient />
            </GradientDefs>
            <HorizontalBarSeries
                colorType="literal"
                barWidth={0.2}
                style={{
                    height: 3,
                    rx: '2px',
                    cursor: 'pointer'
                }}
                color="url(#horizontalGradient)"
                data={formattedData}
                onValueClick={onValueClickHandler}
            />
            <XAxis title={title} />
            <LabelSeries
                className="text-xs text-base-600"
                labelAnchorY="text-top"
                data={labelData}
                onValueClick={onValueClickHandler}
            />
        </FlexibleXYPlot>
    );
};

const HOCLabeledBarGraph = withRouter(LabeledBarGraph);

HOCLabeledBarGraph.propTypes = {
    data: PropTypes.arrayOf(
        PropTypes.shape({
            color: PropTypes.string,
            x: PropTypes.number.isRequired,
            y: PropTypes.number.isRequired,
            url: PropTypes.string
        })
    ),
    title: PropTypes.string
};

HOCLabeledBarGraph.defaultProps = {
    data: [],
    title: null
};

export default HOCLabeledBarGraph;
