import React, { useMemo } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';

import {
    BarChart,
    Bar,
    Cell,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    Legend,
    ResponsiveContainer,
} from 'recharts';
import Slider from 'react-slick';
import slickSettings from 'constants/slickSettings';
import { severityColorMap } from 'constants/severityColors';
import { severityLabels } from 'messages/common';
import NoResultsMessage from 'Components/NoResultsMessage';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import severityPropType from './severityPropTypes';

const ViolationsByClusterChart = ({ history, globalViolationsCounts }) => {
    const violationsByCluster = useMemo(
        () => formatViolationsByCluster(globalViolationsCounts),
        [globalViolationsCounts]
    );
    function makeBarClickHandler(clusterName, severity) {
        return () => {
            const searchFilter = { Severity: severity };
            // if clusters are not loaded yet, at least we can redirect to unfiltered violations
            if (clusterName !== '') {
                searchFilter.Cluster = clusterName;
            }
            const searchString = getUrlQueryStringForSearchFilter(searchFilter);
            history.push(`/main/violations?${searchString}`);
        };
    }
    if (!violationsByCluster || !violationsByCluster.length) {
        return (
            <NoResultsMessage message="No data available. Please ensure your cluster is properly configured." />
        );
    }

    return (
        <div className="p-0 h-64 w-full">
            <Slider {...slickSettings}>
                {violationsByCluster.map((data, index) => (
                    // eslint-disable-next-line react/no-array-index-key
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
                                    bottom: 5,
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
                                        textAnchor: 'end',
                                    }}
                                />
                                <CartesianGrid strokeDasharray="1 1" />
                                <Tooltip
                                    offset={0}
                                    contentStyle={{ backgroundColor: 'var(--base-100)' }}
                                />
                                <Legend wrapperStyle={{ left: 0, width: '100%' }} />
                                {Object.keys(severityLabels).map((severity) => {
                                    const arr = [];
                                    const bar = (
                                        <Bar
                                            name={severityLabels[severity]}
                                            key={severityLabels[severity]}
                                            dataKey={severityLabels[severity]}
                                            fill={severityColorMap[severity]}
                                        >
                                            {data.map((entry) => (
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
    globalViolationsCounts: PropTypes.arrayOf(
        PropTypes.shape({
            counts: PropTypes.arrayOf(
                PropTypes.shape({
                    count: PropTypes.string.isRequired,
                    severity: severityPropType,
                })
            ),
            group: PropTypes.string.isRequired,
        })
    ).isRequired,
    history: PropTypes.shape({
        push: PropTypes.func.isRequired,
    }),
};

ViolationsByClusterChart.defaultProps = {
    history: null,
};

function formatViolationsByCluster(violationsByCluster) {
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
                Low: 0,
            };
            cluster.counts.forEach((d) => {
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

export default withRouter(ViolationsByClusterChart);
