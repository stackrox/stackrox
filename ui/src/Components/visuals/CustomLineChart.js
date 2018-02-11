import React, { Component } from 'react';
import PropTypes from 'prop-types';

import {
    ResponsiveContainer,
    LineChart,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend
} from 'recharts';

class CustomLineChart extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        xAxisDataKey: PropTypes.string.isRequired,
        yAxisDataKey: PropTypes.string.isRequired,
        children: PropTypes.node.isRequired
    };

    constructor(props) {
        super(props);

        this.state = {};
    }

    render() {
        return (
            <ResponsiveContainer>
                <LineChart
                    data={this.props.data}
                    margin={{
                        top: 5,
                        right: 30,
                        left: 20,
                        bottom: 5
                    }}
                >
                    <XAxis dataKey={this.props.xAxisDataKey} />
                    <YAxis
                        domain={[0, 'dataMax']}
                        allowDecimals={false}
                        label={{
                            value: this.props.yAxisDataKey,
                            angle: -90,
                            position: 'insideLeft',
                            textAnchor: 'middle'
                        }}
                    />
                    <CartesianGrid strokeDasharray="3 3" />
                    <Tooltip />
                    <Legend />
                    {this.props.children}
                </LineChart>
            </ResponsiveContainer>
        );
    }
}

export default CustomLineChart;
