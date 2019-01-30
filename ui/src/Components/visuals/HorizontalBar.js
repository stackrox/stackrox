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

const minimalMargin = { top: 0, bottom: 0, left: 0, right: 0 };

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
        valueGradientColorStart: 'var(--tertiary-300)',
        valueGradientColorEnd: 'var(--tertiary-300',
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
        this.props.data.map(item => {
            let label = '';
            // This prevents overlap between the value label and the axis label
            if (item.x > 10) {
                label = this.props.valueFormat(item.x).toString() || '';
            }
            const val = {
                x: 0,
                y: item.y,
                yOffset: 1,
                label
            };
            val.x -= 2.4 * val.label.length;
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
            className: 'relative chart-container w-full',
            onMouseMove: hintsEnabled ? this.setHintPosition : null
        };
        return merge(defaultContainerProps, this.props.containerProps);
    };

    getPlotProps = hintsEnabled => {
        const { data, minimal } = this.props;
        // This determines how far to push the bar graph to the right based on the longest axis label character's length
        const maxLength = data.reduce((acc, curr) => Math.max(curr.y.length, acc), 0);
        const defaultPlotProps = {
            height: minimal ? 30 : 270,
            xDomain: [0, 105],
            yType: 'category',
            yRange: data.map((item, i) => (i + 1) * 23).concat([0]),
            margin: minimal ? minimalMargin : { top: 30, left: Math.ceil(maxLength * 6.4) },
            stackBy: 'x',
            animation: hintsEnabled ? false : ''
        };
        return merge(defaultPlotProps, this.props.plotProps);
    };

    getSeriesProps = () => {
        const defaultSeriesProps = {
            color: 'url(#horizontalGradient)',
            style: {
                height: 12,
                rx: '3px',
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

        const { hintX, hintY, hintData } = this.state;

        const hintsEnabled = !!data.find(item => item.hint);

        // Generate y axis links
        const axisLinks = data.reduce((acc, curr) => {
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
                    <Link className="underline" to={axisLinks[value]}>
                        {value}
                    </Link>
                );

            return <tspan>{inner}</tspan>;
        }

        return (
            <div {...containerProps}>
                {/* Bar Background  */}
                <svg
                    height={`${minimal ? '0' : '10'}`}
                    width="10"
                    xmlns="http://www.w3.org/2000/svg"
                    version="1.1"
                >
                    <defs>
                        <pattern
                            id="bar-background"
                            patternUnits="userSpaceOnUse"
                            width="10"
                            height="10"
                        >
                            background-color: #ffffff;
                            <image
                                xlinkHref="data:image/svg+xml,%3Csvg width='40' height='40' viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='%23ccc9d2' fill-opacity='.3' fill-rule='evenodd'%3E%3Cpath d='M0 40L40 0H20L0 20M40 40V20L20 40'/%3E%3C/g%3E%3C/svg%3E"
                                x="0"
                                y="0"
                                width="10"
                                height="10"
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
                        data={data.map(item => ({ x: 0, x0: 100, y: item.y }))}
                        style={{
                            height: seriesProps.style.height,
                            stroke: 'var(--base-200)',
                            fill: `url(#bar-background)`,
                            cursor: 'pointer'
                        }}
                    />

                    {/* Values */}
                    <HorizontalBarSeries data={data} {...seriesProps} />
                    <LabelSeries
                        data={this.getLabelData()}
                        className="text-xs"
                        labelAnchorY="no-change"
                        labelAnchorX="end-alignment"
                        style={{
                            fill: 'var(--base-800)',
                            cursor: 'pointer'
                        }}
                    />

                    {!minimal && (
                        <YAxis tickSize={0} top={25} className="text-xs" tickFormat={tickFormat} />
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
