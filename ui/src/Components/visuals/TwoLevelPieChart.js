import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { ResponsiveContainer, PieChart, Pie, Sector, Legend, Cell } from 'recharts';

class TwoLevelPieChart extends Component {
    static propTypes = {
        data: PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                value: PropTypes.number.isRequired,
                color: PropTypes.string.isRequired,
                onClick: PropTypes.func.isRequired
            })
        ).isRequired
    };

    constructor(props) {
        super(props);

        this.state = {
            activeIndex: 0
        };
    }

    onPieEnter = (data, index) => {
        this.setState({
            activeIndex: index
        });
    };

    renderActiveShape = properties => {
        const RADIAN = Math.PI / 180;
        const {
            cx,
            cy,
            midAngle,
            innerRadius,
            outerRadius,
            startAngle,
            endAngle,
            fill,
            payload,
            value,
            onClick
        } = properties;
        const sin = Math.sin(-RADIAN * midAngle);
        const cos = Math.cos(-RADIAN * midAngle);
        const sx = cx + (outerRadius + 10) * cos;
        const sy = cy + (outerRadius + 10) * sin;
        const mx = cx + (outerRadius + 30) * cos;
        const my = cy + (outerRadius + 30) * sin;
        const ex = mx + (cos >= 0 ? 1 : -1) * 22;
        const ey = my;
        const textAnchor = cos >= 0 ? 'start' : 'end';
        return (
            <g className="cursor-pointer" onClick={onClick}>
                <text x={cx} y={cy} dy={8} textAnchor="middle" fill="#333">
                    {payload.name}
                </text>
                <Sector
                    cx={cx}
                    cy={cy}
                    innerRadius={innerRadius}
                    outerRadius={outerRadius}
                    startAngle={startAngle}
                    endAngle={endAngle}
                    fill={fill}
                />
                <Sector
                    cx={cx}
                    cy={cy}
                    startAngle={startAngle}
                    endAngle={endAngle}
                    innerRadius={outerRadius + 6}
                    outerRadius={outerRadius + 10}
                    fill={fill}
                />
                <path d={`M${sx},${sy}L${mx},${my}L${ex},${ey}`} stroke={fill} fill="none" />
                <circle cx={ex} cy={ey} r={2} fill={fill} stroke="none" />
                <text
                    x={ex + (cos >= 0 ? 1 : -1) * 12}
                    y={ey}
                    textAnchor={textAnchor}
                    fill="#333"
                >{`${payload.name} (${value})`}</text>
            </g>
        );
    };

    render() {
        return (
            <ResponsiveContainer>
                <PieChart margin={{ top: -20 }}>
                    <Pie
                        startAngle={90}
                        endAngle={500}
                        sAnimationActive={false}
                        activeIndex={this.state.activeIndex}
                        activeShape={this.renderActiveShape}
                        data={this.props.data}
                        dataKey="value"
                        innerRadius={70}
                        outerRadius={80}
                        onMouseEnter={this.onPieEnter}
                    >
                        {this.props.data.map(entry => (
                            <Cell key={entry.name} fill={entry.color} />
                        ))}
                    </Pie>
                    <Legend />
                </PieChart>
            </ResponsiveContainer>
        );
    }
}

export default TwoLevelPieChart;
