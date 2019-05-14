import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { withRouter } from 'react-router-dom';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';

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
import NoResultsMessage from 'Components/NoResultsMessage';

const ViolationsByClusterChart = ({ history, violationsByCluster }) => {
    function makeBarClickHandler(clusterName, severity) {
        return () => {
            // if clusters are not loaded yet, at least we can redirect to unfiltered violations
            const clusterQuery = clusterName !== '' ? `cluster=${clusterName}` : '';
            history.push(`/main/violations?severity=${severity}&${clusterQuery}`);
        };
    }
    if (!violationsByCluster || !violationsByCluster.length) {
        return <NoResultsMessage />;
    }

    return (
        <div className="p-0 h-64 w-full">
            <Slider {...slickSettings}>
                {violationsByCluster.map((data, index) => (
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
                                <Tooltip
                                    offset={0}
                                    contentStyle={{ backgroundColor: 'var(--base-100)' }}
                                />
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
                                                    onClick={makeBarClickHandler(
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
};

ViolationsByClusterChart.propTypes = {
    violationsByCluster: PropTypes.arrayOf(
        PropTypes.arrayOf(
            PropTypes.shape({
                name: PropTypes.string.isRequired,
                Critical: PropTypes.number.isRequired,
                High: PropTypes.number.isRequired,
                Medium: PropTypes.number.isRequired,
                Low: PropTypes.number.isRequired
            })
        )
    ),
    history: PropTypes.shape({
        push: PropTypes.func.isRequired
    })
};

ViolationsByClusterChart.defaultProps = {
    violationsByCluster: [],
    history: null
};

const formatViolationsByCluster = createSelector(
    [selectors.getAlertCountsByCluster],
    violationsByCluster => {
        const clusterCharts = [];

        let i = 0;
        const limit = 4;
        while (i < violationsByCluster.length) {
            let j = i;
            let groupIndex = 0;
            const barCharts = [];
            while (j < violationsByCluster.length && groupIndex < limit) {
                const cluster = violationsByCluster[j];
                const dataPoint = {
                    name: cluster.group,
                    Critical: 0,
                    High: 0,
                    Medium: 0,
                    Low: 0
                };
                cluster.counts.forEach(d => {
                    dataPoint[severityLabels[d.severity]] = parseInt(d.count, 10);
                });
                barCharts.push(dataPoint);
                j += 1;
                groupIndex += 1;
            }
            clusterCharts.push(barCharts);
            i += 4;
        }
        return clusterCharts;
    }
);

const mapStateToProps = createStructuredSelector({
    violationsByCluster: formatViolationsByCluster
});

export default withRouter(
    connect(
        mapStateToProps,
        null
    )(ViolationsByClusterChart)
);
