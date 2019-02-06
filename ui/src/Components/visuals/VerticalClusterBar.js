import React, { Component } from 'react';
import {
    FlexibleXYPlot,
    XAxis,
    YAxis,
    VerticalGridLines,
    HorizontalGridLines,
    VerticalBarSeries
} from 'react-vis';
import PropTypes from 'prop-types';
import { withRouter, Link } from 'react-router-dom';
import DiscreteColorLegend from 'react-vis/dist/legends/discrete-color-legend';
import merge from 'deepmerge';
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
        onValueMouseOut: PropTypes.func
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
        onValueMouseOut: null
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
        return Object.keys(data)
            .sort()
            .map((key, i) => ({
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
            onValueMouseOut
        } = this.props;

        // Default props
        const defaultPlotProps = {
            xType: 'ordinal',
            yDomain: [0, 160]
        };

        const defaultContainerProps = {
            className: 'relative chart-container w-full',
            onMouseMove: this.setHintPosition
        };

        const defaultSeriesProps = {
            // animation: true, //causes onValueMouseOut to fail https://github.com/uber/react-vis/issues/381
            barWidth: 0.5,
            style: {
                opacity: '.8',
                width: '6px',
                ry: '2px',
                cursor: 'pointer'
            },
            onValueMouseOver: datum => {
                this.setHintData(datum);
                if (onValueMouseOver) onValueMouseOver(datum);
            },
            onValueMouseOut: datum => {
                this.clearHintData();
                if (onValueMouseOut) onValueMouseOut(datum);
            },
            onValueClick: datum => {
                if (datum.link) this.props.history.push(datum.link);
            }
        };

        // Merge props
        const containerProps = merge(defaultContainerProps, this.props.containerProps);
        const plotProps = merge(defaultPlotProps, this.props.plotProps);
        const seriesProps = merge(defaultSeriesProps, this.props.seriesProps);

        function formatTicks(value) {
            let inner = value;
            if (labelLinks[value])
                inner = (
                    <Link className="underline" to={labelLinks[value]}>
                        {value}
                    </Link>
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
        Object.keys(data)
            .sort()
            .forEach((key, i) => {
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
                <div className="flex flex-col h-full">
                    <FlexibleXYPlot {...plotProps}>
                        <VerticalGridLines
                            left={330 / clusterNames.length / 2 + 30}
                            tickValues={clusterNames.slice(0, clusterNames.length - 1)}
                        />
                        <HorizontalGridLines tickValues={tickValues} />
                        <YAxis tickValues={tickValues} tickSize={0} tickFormat={tickFormat} />
                        {series}

                        <XAxis tickSize={0} tickFormat={formatTicks} />
                    </FlexibleXYPlot>
                    <div>
                        <DiscreteColorLegend
                            orientation="horizontal"
                            items={this.getLegendData()}
                            colors={colors}
                            className="horizontal-bar-legend"
                        />
                    </div>
                    {this.state.hintData ? (
                        <HoverHint
                            top={this.state.hintY}
                            left={this.state.hintX}
                            title={this.state.hintData.title}
                            body={this.state.hintData.body}
                        />
                    ) : null}
                </div>
            </div>
        );
    }
}

export default withRouter(BarChart);
