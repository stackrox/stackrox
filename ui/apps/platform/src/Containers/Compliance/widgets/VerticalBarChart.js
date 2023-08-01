import React, { Component } from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    VerticalBarSeries,
} from 'react-vis';
import PropTypes from 'prop-types';
import merge from 'deepmerge';

import { verticalBarColors } from './colorsForCompliance';

const sortByXValue = (a, b) => {
    if (a.x < b.x) {
        return -1;
    }
    if (a.x > b.x) {
        return 1;
    }
    return 0;
};

// Component state was needed only for deleted HoverHint, but we will not rewrite.
/* eslint-disable react/prefer-stateless-function */
class VerticalBarChart extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.object).isRequired,
        containerProps: PropTypes.shape({}),
        plotProps: PropTypes.shape({}),
        seriesProps: PropTypes.shape({}),
        tickValues: PropTypes.arrayOf(PropTypes.number),
        tickFormat: PropTypes.func,
        labelLinks: PropTypes.shape({}),
        onValueClick: PropTypes.func,
        legend: PropTypes.bool,
    };

    static defaultProps = {
        containerProps: {},
        plotProps: {},
        seriesProps: {},
        tickValues: [25, 50, 75, 100],
        tickFormat: (x) => `${x}%`,
        labelLinks: {},
        onValueClick: null,
        legend: true,
    };

    render() {
        const { tickValues, tickFormat, labelLinks, onValueClick } = this.props;

        const data = this.props.data.sort(sortByXValue);

        // Default props
        const defaultPlotProps = {
            xType: 'ordinal',
            height: 250,
            colorType: 'category',
            yDomain: [0, 110],
        };

        const defaultContainerProps = {
            className: 'relative chart-container w-full horizontal-bar-responsive',
        };

        const defaultSeriesProps = {
            barWidth: 0.25,
            style: {
                opacity: '.8',
                ry: '2px',
                cursor: 'pointer',
            },

            colorDomain: data.map((datum) => datum.y),
            colorRange: verticalBarColors,
            onValueClick: (datum) => {
                if (onValueClick) {
                    onValueClick(datum);
                }
            },
        };

        // Merge props
        const containerProps = merge(defaultContainerProps, this.props.containerProps);
        const plotProps = merge(defaultPlotProps, this.props.plotProps);
        const seriesProps = merge(defaultSeriesProps, this.props.seriesProps);
        const styleProps = this.props.legend ? { top: '-16px' } : {};

        // format data with colors:
        const letDataWithColors = data.map((datum, i) => ({
            ...datum,
            color: verticalBarColors[i % verticalBarColors.length],
        }));

        function formatTicks(value) {
            let inner = value;
            if (labelLinks[value]) {
                inner = (
                    <a className="underline" href={labelLinks[value]}>
                        {value}
                    </a>
                );
            }

            return <tspan>{inner}</tspan>;
        }

        return (
            <div style={styleProps} {...containerProps}>
                <FlexibleWidthXYPlot {...plotProps}>
                    <VerticalGridLines left={330 / data.length / 2 + 30} />
                    <HorizontalGridLines tickValues={tickValues} />
                    <YAxis tickValues={tickValues} tickSize={0} tickFormat={tickFormat} />
                    <VerticalBarSeries data={letDataWithColors} {...seriesProps} />
                    <XAxis tickSize={0} tickFormat={formatTicks} />
                </FlexibleWidthXYPlot>
            </div>
        );
    }
}

export default VerticalBarChart;
