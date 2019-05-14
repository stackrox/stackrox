import React from 'react';
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

const CustomLineChart = ({ data, name, xAxisDataKey, children }) => (
    <ResponsiveContainer>
        <LineChart
            data={data}
            margin={{
                top: 30,
                right: 50,
                left: 10,
                bottom: 10
            }}
        >
            <XAxis dataKey={xAxisDataKey}>
                <Label value={name} fill="var(--primary-800)" offset={180} position="top" />
            </XAxis>
            <YAxis domain={[0, 'dataMax']} allowDecimals={false} />
            <CartesianGrid strokeDasharray="1 1" />
            <Tooltip offset={0} contentStyle={{ backgroundColor: 'var(--base-100)' }} />
            <Legend
                wrapperStyle={{
                    left: 0,
                    bottom: 0,
                    width: '100%',
                    textTransform: 'capitalize',
                    fill: 'var(--primary-800)'
                }}
            />
            {children}
        </LineChart>
    </ResponsiveContainer>
);

CustomLineChart.propTypes = {
    data: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    name: PropTypes.string.isRequired,
    xAxisDataKey: PropTypes.string.isRequired,
    children: PropTypes.node.isRequired
};

export default CustomLineChart;
