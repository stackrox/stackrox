import React, { Component } from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalBarSeries,
    GradientDefs,
    LabelSeries,
} from 'react-vis';
import { withRouter, Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import merge from 'deepmerge';

import BarGradient from 'Components/visuals/BarGradient';

const minimalMargin = { top: -15, bottom: 0, left: 0, right: 0 };

const sortByYValue = (a, b) => {
    if (a.y < b.y) {
        return -1;
    }
    if (a.y > b.y) {
        return 1;
    }
    return 0;
};

class HorizontalBarChart extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        containerProps: PropTypes.shape({}),
        plotProps: PropTypes.shape({}),
        seriesProps: PropTypes.shape({}),
        valueFormat: PropTypes.func,
        tickValues: PropTypes.arrayOf(PropTypes.number),
        valueGradientColorStart: PropTypes.string,
        valueGradientColorEnd: PropTypes.string,
        minimal: PropTypes.bool,
        history: ReactRouterPropTypes.history.isRequired,
    };

    static defaultProps = {
        valueFormat: (x) => x,
        tickValues: [0, 25, 50, 75, 100],
        containerProps: {},
        plotProps: {},
        seriesProps: {},
        valueGradientColorStart: '#B3DCFF',
        valueGradientColorEnd: '#BDF3FF',
        minimal: false,
    };

    showLabel = (value) => value >= 10;

    getLabelData = () =>
        this.props.data.sort(sortByYValue).map((item) => {
            let label = '';
            // This prevents overlap between the value label and the axis label
            if (this.showLabel(item.x)) {
                label = (this.props.valueFormat && this.props.valueFormat(item.x).toString()) || '';
            }
            const { minimal } = this.props;
            const val = {
                x: item.x - 5,
                y: item.y,
                yOffset: minimal ? -6 : -3,
                label,
            };
            // x offset for label
            val.x -= val.label.length;
            return val;
        });

    onValueClickHandler = (datum) => {
        if (datum.link) {
            this.props.history.push(datum.link);
        }
    };

    getContainerProps = () => {
        const defaultContainerProps = {
            className: 'relative chart-container w-full horizontal-bar-responsive',
        };
        return merge(defaultContainerProps, this.props.containerProps);
    };

    getPlotProps = () => {
        const { minimal, data } = this.props;
        const sortedData = data.sort(sortByYValue);
        // This determines how far to push the bar graph to the right based on the longest axis label character's length
        const maxLength = sortedData.reduce((acc, curr) => Math.max(curr.y.length, acc), 0);

        // Magic number that makes the horizontal bars a reasonable size if there are fewer than 6
        // but shrinks them if there are 6 or more.
        const yRangeMultiplier = Math.min(41, 210 / sortedData.length);
        const yRange = [...sortedData.map((item, i) => (i + 1) * yRangeMultiplier), 0];
        const defaultPlotProps = {
            height: minimal ? 25 : 350,
            xDomain: [0, 102],
            yType: 'category',
            yRange,
            margin: minimal ? minimalMargin : { top: 33.3, left: Math.ceil(maxLength * 7.5) },
            stackBy: 'x',
        };
        return merge(defaultPlotProps, this.props.plotProps);
    };

    getSeriesProps = () => {
        const { minimal } = this.props;
        const defaultSeriesProps = {
            color: 'url(#horizontalGradient)',
            style: {
                height: minimal ? 15 : 20,
                rx: '2px',
                cursor: `${this.props.minimal ? '' : 'pointer'}`,
            },
            onValueMouseOver: this.onValueMouseOverHandler,
            onValueMouseOut: this.onValueMouseOutHandler,
            onValueClick: this.onValueClickHandler,
        };
        return merge(defaultSeriesProps, this.props.seriesProps);
    };

    render() {
        const { data, tickValues, valueFormat, minimal } = this.props;
        const sortedData = data.sort(sortByYValue);

        // Generate y axis links
        const axisLinks = sortedData.reduce((acc, curr) => {
            if (curr.link) {
                acc[curr.y] = curr.link;
            }
            return acc;
        }, {});

        const containerProps = this.getContainerProps();
        const plotProps = this.getPlotProps();
        const seriesProps = this.getSeriesProps();

        function tickFormat(value) {
            let inner = value;
            if (axisLinks[value]) {
                inner = (
                    <Link
                        style={{ fill: 'currentColor' }}
                        className="underline text-sm text-base-600"
                        to={axisLinks[value]}
                    >
                        {value}
                    </Link>
                );
            }

            return <tspan>{inner}</tspan>;
        }

        return (
            <div {...containerProps}>
                {/* Bar Background  */}
                <svg
                    className="absolute"
                    height="0"
                    width="10"
                    xmlns="http://www.w3.org/2000/svg"
                    version="1.1"
                >
                    <defs>
                        <pattern
                            id="bar-background"
                            patternUnits="userSpaceOnUse"
                            width="6"
                            height="6"
                        >
                            background-color: #ffffff;
                            <image
                                xlinkHref="data:image/svg+xml,%3Csvg width='6' height='6' viewBox='0 0 6 6' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='%23ccc9d2' fill-opacity='0.4' fill-rule='evenodd'%3E%3Cpath d='M5 0h1L0 6V5zM6 5v1H5z'/%3E%3C/g%3E%3C/svg%3E"
                                x="0"
                                y="0"
                                width="6"
                                height="6"
                            />
                        </pattern>
                    </defs>
                </svg>
                <FlexibleWidthXYPlot {...plotProps}>
                    <GradientDefs>
                        <BarGradient />
                    </GradientDefs>
                    {/* Empty area bar background */}

                    {!minimal && <VerticalGridLines tickValues={tickValues} />}

                    {!minimal && (
                        <XAxis
                            orientation="top"
                            tickSize={0}
                            tickFormat={valueFormat}
                            tickValues={tickValues}
                        />
                    )}

                    {/* Empty Background */}
                    <HorizontalBarSeries
                        data={sortedData.map((item) => ({
                            x: 0,
                            x0: Math.ceil(sortedData[0].x / 5) * 5,
                            y: item.y,
                            link: item.link,
                        }))}
                        style={{
                            height: seriesProps.style.height,
                            stroke: 'var(--base-300)',
                            fill: `url(#bar-background)`,
                            rx: '2',
                            ry: '2',
                            cursor: `${minimal ? '' : 'pointer'}`,
                        }}
                        onValueClick={this.onValueClickHandler}
                    />

                    {/* Values */}
                    <HorizontalBarSeries data={sortedData} {...seriesProps} />
                    <LabelSeries
                        data={this.getLabelData()}
                        className="text-xs pointer-events-none theme-light"
                        labelAnchorY="no-change"
                        labelAnchorX="end-alignment"
                        style={{
                            fill: 'var(--primary-800)',
                            cursor: `${minimal ? '' : 'pointer'}`,
                        }}
                    />

                    {!minimal && (
                        <YAxis tickSize={0} top={26} className="text-xs" tickFormat={tickFormat} />
                    )}
                </FlexibleWidthXYPlot>
            </div>
        );
    }
}

export default withRouter(HorizontalBarChart);
