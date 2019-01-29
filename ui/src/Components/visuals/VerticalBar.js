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
import merge from 'deepmerge';
import HoverHint from './HoverHint';

class VerticalBarChart extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.object).isRequired,
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
            'var(--accent-500)',
            'var(--success-500)',
            'var(--tertiary-500)'
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
            colorType: 'category',
            yDomain: [0, 110]
        };

        const defaultContainerProps = {
            className: 'relative chart-container',
            onMouseMove: this.setHintPosition
        };

        const defaultSeriesProps = {
            // animation: true, //causes onValueMouseOut to fail https://github.com/uber/react-vis/issues/381
            barWidth: 0.28,
            style: {
                opacity: '.8',
                ry: '2px',
                cursor: 'pointer'
            },

            colorDomain: data.map(datum => datum.y),
            colorRange: colors,
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

        // Merge props
        const containerProps = merge(defaultContainerProps, this.props.containerProps);
        const plotProps = merge(defaultPlotProps, this.props.plotProps);
        const seriesProps = merge(defaultSeriesProps, this.props.seriesProps);

        // format data with colors:
        const letDataWithColors = data.map((datum, i) =>
            Object.assign({}, datum, { color: colors[i % colors.length] })
        );

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

        return (
            <div {...containerProps}>
                <FlexibleWidthXYPlot {...plotProps}>
                    <VerticalGridLines left={330 / data.length / 2 + 30} />
                    <HorizontalGridLines tickValues={tickValues} />
                    <YAxis tickValues={tickValues} tickSize={0} tickFormat={tickFormat} />
                    <VerticalBarSeries data={letDataWithColors} {...seriesProps} />
                    <XAxis tickSize={0} tickFormat={formatTicks} />
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

export default VerticalBarChart;
