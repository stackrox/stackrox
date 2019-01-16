import React, { Component } from 'react';
import PropTypes from 'prop-types';
import {
    BarChart,
    Bar,
    Cell,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer
} from 'recharts';
import Slider from 'react-slick';
import slickSettings from 'constants/slickSettings';
import severityColorMap from 'constants/severityColors';
import { severityLabels } from 'messages/common';

class ViolationsByClusterChart extends Component {
    static propTypes = {
        clusterCharts: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        history: PropTypes.shape({
            push: PropTypes.func.isRequired
        }).isRequired
    };

    makeBarClickHandler = (clusterName, severity) => () => {
        // if clusters are not loaded yet, at least we can redirect to unfiltered violations
        const clusterQuery = clusterName !== '' ? `cluster=${clusterName}` : '';
        this.props.history.push(`/main/violations?severity=${severity}&${clusterQuery}`);
    };

    render() {
        const { clusterCharts } = this.props;
        return (
            <div className="p-0 h-64 w-full">
                <Slider {...slickSettings}>
                    {clusterCharts.map((data, index) => (
                        <div key={index}>
                            <ResponsiveContainer className="flex-1 h-full w-full">
                                <BarChart
                                    stackOffset="expand"
                                    maxBarSize={32}
                                    barGap={16}
                                    data={data}
                                    margin={{
                                        top: 5,
                                        right: 10,
                                        left: -30,
                                        bottom: 5
                                    }}
                                >
                                    <XAxis dataKey="name" />

                                    <YAxis
                                        domain={[0, 'dataMax']}
                                        allowDecimals={false}
                                        label={{
                                            value: '',
                                            angle: -90,
                                            position: 'insideLeft',
                                            textAnchor: 'end'
                                        }}
                                    />
                                    <CartesianGrid strokeDasharray="1 1" />
                                    <Tooltip offset={0} />
                                    <Legend wrapperStyle={{ left: 0, width: '100%' }} />
                                    {Object.keys(severityLabels).map(severity => {
                                        const arr = [];
                                        const bar = (
                                            <Bar
                                                name={severityLabels[severity]}
                                                key={severityLabels[severity]}
                                                dataKey={severityLabels[severity]}
                                                fill={severityColorMap[severity]}
                                            >
                                                {data.map(entry => (
                                                    <Cell
                                                        key={entry.name}
                                                        className="cursor-pointer"
                                                        onClick={this.makeBarClickHandler(
                                                            entry.name,
                                                            severity
                                                        )}
                                                    />
                                                ))}
                                            </Bar>
                                        );
                                        arr.push(bar);
                                        return arr;
                                    })}
                                </BarChart>
                            </ResponsiveContainer>
                        </div>
                    ))}
                </Slider>
            </div>
        );
    }
}

ViolationsByClusterChart.propTypes = {
    clusterCharts: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    history: PropTypes.shape({
        push: PropTypes.func.isRequired
    }).isRequired
};

export default ViolationsByClusterChart;
