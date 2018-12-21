import React, { Component } from 'react';
import {
    FlexibleWidthXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    VerticalBarSeries
} from 'react-vis';

import PropTypes from 'prop-types';
import DiscreteColorLegend from 'react-vis/dist/legends/discrete-color-legend';
import HoverHint from './HoverHint';

class BarChart extends Component {
    static propTypes = {
        data: PropTypes.shape({}).isRequired,
        colors: PropTypes.arrayOf(PropTypes.string),
        containerProps: PropTypes.shape({}),
        plotProps: PropTypes.shape({}),
        seriesProps: PropTypes.shape({}),
        tickValues: PropTypes.arrayOf(PropTypes.number),
        tickFormat: PropTypes.func,
        labelLinks: PropTypes.shape({}),
        onValueMouseOver: PropTypes.func,
        onValueMouseOut: PropTypes.func,
        onValueClick: PropTypes.func
    };

    static defaultProps = {
        colors: [
            'var(--primary-500)',
            'var(--secondary-500)',
            'var(--tertiary-500)',
            'var(--accent-500)'
        ],
        containerProps: {},
        plotProps: {},
        seriesProps: {},
        tickValues: [25, 50, 75, 100],
        tickFormat: x => `${x}%`,
        labelLinks: {},
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
        this.setState({
            hintData: val.hint
        });
    };

    setHintPosition = ev => {
        const container = ev.target.closest('.relative').getBoundingClientRect();
        const offset = 10;
        this.setState({
            hintX: ev.clientX - container.left + offset,
            hintY: ev.clientY - container.top + offset
        });
    };

    clearHintData = () => {
        this.setState({ hintData: null });
    };

    getLegendData = () => {
        const { data, colors } = this.props;
        return Object.keys(data).map((key, i) => ({
            title: key,
            color: colors[i % colors.length]
        }));
    };

    render() {
        const {
            data,
            colors,
            tickValues,
            tickFormat,
            labelLinks,
            onValueMouseOver,
            onValueMouseOut,
            onValueClick
        } = this.props;

        // Default props
        const defaultPlotProps = {
            xType: 'ordinal',
            height: 250,
            yDomain: [0, 110]
        };

        const defaultContainerProps = {
            className: 'relative',
            onMouseMove: this.setHintPosition
        };

        const defaultSeriesProps = {
            // animation: true, //causes onValueMouseOut to fail https://github.com/uber/react-vis/issues/381
            barWidth: 0.5,
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
                opacity: '.8',
                width: '6px',
                ry: '2px'
            },
            this.props.seriesProps.style
        );

        // Merge props
        const containerProps = Object.assign({}, defaultContainerProps, this.props.containerProps);
        const plotProps = Object.assign({}, defaultPlotProps, this.props.plotProps);
        const seriesProps = Object.assign({}, defaultSeriesProps, this.props.seriesProps);
        seriesProps.style = seriesStyle;

        function formatTicks(value) {
            let inner = value;
            if (labelLinks[value])
                inner = (
                    <a className="underline" href={labelLinks[value]}>
                        {value}
                    </a>
                );

            return <tspan>{inner}</tspan>;
        }

        // Calculate unique cluster names
        let clusterNames = new Set();
        Object.keys(data).forEach(dataSetKey => {
            const dataSet = data[dataSetKey];
            dataSet.forEach(datum => {
                clusterNames.add(datum.x);
            });
        });
        clusterNames = Array.from(clusterNames);

        // Create Barseries for each data set
        const series = [];
        Object.keys(data).forEach((key, i) => {
            series.push(
                <VerticalBarSeries
                    data={data[key]}
                    color={colors[i % colors.length]}
                    {...seriesProps}
                    key={key}
                />
            );
        });

        return (
            <div {...containerProps}>
                <FlexibleWidthXYPlot {...plotProps}>
                    <VerticalGridLines
                        left={330 / clusterNames.length / 2 + 30}
                        tickValues={clusterNames.slice(0, clusterNames.length - 1)}
                    />
                    <HorizontalGridLines tickValues={tickValues} />
                    <YAxis tickValues={tickValues} tickSize={0} tickFormat={tickFormat} />
                    {series}

                    <XAxis tickSize={0} tickFormat={formatTicks} />
                    <DiscreteColorLegend
                        orientation="horizontal"
                        items={this.getLegendData()}
                        colors={colors}
                        className="horizontal-bar-legend"
                    />
                </FlexibleWidthXYPlot>
                {this.state.hintData ? (
                    <HoverHint
                        top={this.state.hintY}
                        left={this.state.hintX}
                        title={this.state.hintData.title}
                        body={this.state.hintData.body}
                    />
                ) : null}
            </div>
        );
    }
}

export default BarChart;
