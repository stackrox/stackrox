import React, { Component } from 'react';
import PropTypes from 'prop-types';

import {
    ResponsiveContainer,
    LineChart,
    Label,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend
} from 'recharts';

class CustomLineChart extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
        name: PropTypes.string.isRequired,
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
                        top: 30,
                        right: 50,
                        left: 10,
                        bottom: 10
                    }}
                >
                    <XAxis dataKey={this.props.xAxisDataKey}>
                        <Label value={this.props.name} fill="#696e89" offset={180} position="top" />
                    </XAxis>
                    <YAxis
                        domain={[0, 'dataMax']}
                        allowDecimals={false}
                        label={{
                            value: this.props.yAxisDataKey,
                            angle: -90,
                            fill: '#696e89',
                            position: 'insideLeft',
                            textAnchor: 'middle'
                        }}
                    />
                    <CartesianGrid strokeDasharray="1 1" />
                    <Tooltip offset={0} />
                    <Legend
                        wrapperStyle={{
                            left: 0,
                            bottom: 0,
                            width: '100%',
                            textTransform: 'capitalize'
                        }}
                    />
                    {this.props.children}
                </LineChart>
            </ResponsiveContainer>
        );
    }
}

export default CustomLineChart;
