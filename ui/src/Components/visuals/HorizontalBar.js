import React, { Component } from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalBarSeries,
    GradientDefs,
    LabelSeries
} from 'react-vis';
import { withRouter, Link } from 'react-router-dom';
import PropTypes from 'prop-types';
import merge from 'deepmerge';

import HoverHint from './HoverHint';

const minimalMargin = { top: -15, bottom: 0, left: 0, right: 0 };

const sortByYValue = (a, b) => {
    if (a.y < b.y) return -1;
    if (a.y > b.y) return 1;
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
        onValueMouseOver: PropTypes.func,
        onValueMouseOut: PropTypes.func,
        minimal: PropTypes.bool
    };

    static defaultProps = {
        valueFormat: x => x,
        tickValues: [0, 25, 50, 75, 100],
        containerProps: {},
        plotProps: {},
        seriesProps: {},
        valueGradientColorStart: '#B3DCFF',
        valueGradientColorEnd: '#BDF3FF',
        onValueMouseOver: null,
        onValueMouseOut: null,
        minimal: false
    };

    constructor(props) {
        super(props);
        this.state = {
            hintData: null
        };
    }

    showLabel = value => value >= 7;

    setHintData = val => {
        if (val.hint) {
            this.setState({
                hintData: val.hint
            });
        }
    };

    clearHintData = () => {
        this.setState({ hintData: null });
    };

    setHintPosition = ev => {
        const container = ev.target.closest('.relative').getBoundingClientRect();
        const offset = 10;
        this.setState({
            hintX: ev.clientX - container.left + offset,
            hintY: ev.clientY - container.top + offset
        });
    };

    getLabelData = () =>
        this.props.data.sort(sortByYValue).map(item => {
            let label = '';
            // This prevents overlap between the value label and the axis label
            if (this.showLabel(item.x)) {
                label = this.props.valueFormat(item.x).toString() || '';
            }
            const { minimal } = this.props;
            const val = {
                x: minimal ? -14 : -3.2,
                y: item.y,
                yOffset: minimal ? -6 : -3,
                label
            };
            // x offset for label
            val.x -= val.label.length;
            return val;
        });

    onValueMouseOverHandler = datum => {
        const { onValueMouseOver } = this.props;
        this.setHintData(datum);
        if (onValueMouseOver) onValueMouseOver(datum);
    };

    onValueMouseOutHandler = datum => {
        const { onValueMouseOut } = this.props;
        this.clearHintData();
        if (onValueMouseOut) onValueMouseOut(datum);
    };

    onValueClickHandler = datum => {
        if (datum.barLink) this.props.history.push(datum.barLink);
    };

    getContainerProps = hintsEnabled => {
        const defaultContainerProps = {
            className: 'relative chart-container w-full horizontal-bar-responsive',
            onMouseMove: hintsEnabled ? this.setHintPosition : null
        };
        return merge(defaultContainerProps, this.props.containerProps);
    };

    getPlotProps = hintsEnabled => {
        const { minimal, data } = this.props;
        const sortedData = data.sort(sortByYValue);
        // This determines how far to push the bar graph to the right based on the longest axis label character's length
        const maxLength = sortedData.reduce((acc, curr) => Math.max(curr.y.length, acc), 0);
        const defaultPlotProps = {
            height: minimal ? 25 : 350,
            xDomain: [0, 102],
            yType: 'category',
            yRange: sortedData.map((item, i) => (i + 1) * 41).concat([0]),
            margin: minimal ? minimalMargin : { top: 33.3, left: Math.ceil(maxLength * 7.5) },
            stackBy: 'x',
            animation: hintsEnabled ? false : ''
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
                cursor: 'pointer'
            },
            onValueMouseOver: this.onValueMouseOverHandler,
            onValueMouseOut: this.onValueMouseOutHandler,
            onValueClick: this.onValueClickHandler
        };
        return merge(defaultSeriesProps, this.props.seriesProps);
    };

    render() {
        const {
            data,
            tickValues,
            valueFormat,
            valueGradientColorStart,
            valueGradientColorEnd,
            minimal
        } = this.props;

        const sortedData = data.sort(sortByYValue);

        const { hintX, hintY, hintData } = this.state;

        const hintsEnabled = !!sortedData.find(item => item.hint);

        // Generate y axis links
        const axisLinks = sortedData.reduce((acc, curr) => {
            if (curr.axisLink) acc[curr.y] = curr.axisLink;
            return acc;
        }, {});

        const containerProps = this.getContainerProps(hintsEnabled);
        const plotProps = this.getPlotProps(hintsEnabled);
        const seriesProps = this.getSeriesProps();

        function tickFormat(value) {
            let inner = value;
            if (axisLinks[value])
                inner = (
                    <Link
                        style={{ fill: 'currentColor' }}
                        className="underline text-2xs font-800 text-base-600 hover:text-primary-700"
                        to={axisLinks[value]}
                    >
                        {value}
                    </Link>
                );

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
                        <linearGradient
                            id="horizontalGradient"
                            gradientUnits="userSpaceOnUse"
                            x1="0"
                            y1="50%"
                            x2="50%"
                            y2="50%"
                        >
                            <stop offset="0%" stopColor={valueGradientColorStart} />
                            <stop offset="100%" stopColor={valueGradientColorEnd} />
                        </linearGradient>
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
                        data={sortedData.map(item => ({
                            x: 0,
                            x0: 100,
                            y: item.y,
                            barLink: item.barLink
                        }))}
                        style={{
                            height: seriesProps.style.height,
                            stroke: 'var(--base-300)',
                            fill: `url(#bar-background)`,
                            rx: '2',
                            ry: '2',
                            cursor: 'pointer'
                        }}
                        onValueClick={this.onValueClickHandler}
                    />

                    {/* Values */}
                    <HorizontalBarSeries data={sortedData} {...seriesProps} />
                    <LabelSeries
                        data={this.getLabelData()}
                        className="text-xs pointer-events-none"
                        labelAnchorY="no-change"
                        labelAnchorX="end-alignment"
                        style={{
                            fill: 'var(--base-700)',

                            cursor: 'pointer'
                        }}
                    />

                    {!minimal && (
                        <YAxis tickSize={0} top={26} className="text-xs" tickFormat={tickFormat} />
                    )}
                </FlexibleWidthXYPlot>

                {hintData && (
                    <HoverHint
                        top={hintY}
                        left={hintX}
                        title={hintData.title}
                        body={hintData.body}
                    />
                )}
            </div>
        );
    }
}

export default withRouter(HorizontalBarChart);
