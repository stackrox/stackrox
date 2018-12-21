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

import PropTypes from 'prop-types';
import HoverHint from './HoverHint';

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
        onValueClick: PropTypes.func
    };

    static defaultProps = {
        valueFormat: x => x,
        tickValues: [0, 25, 50, 75, 100],
        containerProps: null,
        plotProps: null,
        seriesProps: null,
        valueGradientColorStart: 'var(--tertiary-500)',
        valueGradientColorEnd: 'var(--tertiary-400',
        onValueMouseOver: null,
        onValueMouseOut: null,
        onValueClick: null
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
            const val = {
                x: 0,
                y: item.y,
                yOffset: 1,
                label: this.props.valueFormat(item.x).toString() || ''
            };
            val.x -= 2.4 * val.label.length;
            return val;
        });

    render() {
        const {
            data,
            tickValues,
            valueFormat,
            onValueMouseOver,
            onValueMouseOut,
            onValueClick,
            valueGradientColorStart,
            valueGradientColorEnd
        } = this.props;

        const { hintX, hintY, hintData } = this.state;

        const hintsEnabled = !!data.find(item => item.hint);

        // Generate y axis links
        const axisLinks = data.reduce((acc, curr) => {
            if (curr.axisLink) acc[curr.y] = curr.axisLink;
            return acc;
        }, {});

        // Default props
        const defaultContainerProps = {
            className: `flex flex-col justify-between h-full ${hintsEnabled ? 'relative' : ''}`,
            onMouseMove: hintsEnabled ? this.setHintPosition : null
        };
        const defaultPlotProps = {
            height: 260,
            xDomain: [0, 105],
            yType: 'category',
            yRange: data.map((item, i) => (i + 1) * 20),
            margin: { top: 30 },
            stackBy: 'x',
            animation: hintsEnabled ? false : ''
        };

        const defaultSeriesProps = {
            color: 'url(#horizontalGradient)',
            onValueMouseOver: datum => {
                this.setHintData(datum);
                if (onValueMouseOver) onValueMouseOver(datum);
            },
            onValueMouseOut: datum => {
                this.clearHintData();
                if (onValueMouseOut) onValueMouseOut(datum);
            },
            onValueClick: datum => {
                if (onValueClick) onValueClick(datum);
            }
        };
        const seriesStyle = Object.assign(
            {
                height: 12,
                rx: '3px'
            },
            this.props.seriesProps && this.props.seriesProps.style
        );

        const containerProps = Object.assign({}, defaultContainerProps, this.props.containerProps);
        const plotProps = Object.assign({}, defaultPlotProps, this.props.plotProps);
        const seriesProps = Object.assign({}, defaultSeriesProps, this.props.seriesProps);
        seriesProps.style = seriesStyle;

        function tickFormat(value) {
            let inner = value;
            if (axisLinks[value])
                inner = (
                    <a className="underline" href={axisLinks[value]}>
                        {value}
                    </a>
                );

            return <tspan>{inner}</tspan>;
        }

        return (
            <div {...containerProps}>
                {/* Bar Background  */}
                <svg height="10" width="10" xmlns="http://www.w3.org/2000/svg" version="1.1">
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

                    <VerticalGridLines tickValues={tickValues} />

                    <XAxis
                        orientation="top"
                        tickSize={0}
                        tickFormat={valueFormat}
                        tickValues={tickValues}
                    />

                    {/* Empty Background */}
                    <HorizontalBarSeries
                        data={data.map(item => ({ x: 0, x0: 100, y: item.y }))}
                        style={{
                            height: seriesProps.style.height,
                            stroke: 'var(--base-200)',
                            fill: `url(#bar-background)`
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
                            fill: 'var(--base-800)'
                        }}
                    />

                    <YAxis tickSize={0} top={25} className="text-xs" tickFormat={tickFormat} />
                </FlexibleWidthXYPlot>

                {hintData ? (
                    <HoverHint
                        top={hintY}
                        left={hintX}
                        title={hintData.title}
                        body={hintData.body}
                    />
                ) : null}
            </div>
        );
    }
}

export default HorizontalBarChart;
