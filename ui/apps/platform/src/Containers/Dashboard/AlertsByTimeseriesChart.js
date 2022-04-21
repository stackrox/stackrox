import React from 'react';
import PropTypes from 'prop-types';

import { Line } from 'recharts';
import Slider from 'react-slick';
import NoResultsMessage from 'Components/NoResultsMessage';
import CustomLineChart from 'Components/visuals/CustomLineChart';
import { severityColorMap } from 'constants/severityColors';
import slickSettings from 'constants/slickSettings';
import cloneDeep from 'lodash/cloneDeep';
import { format, subDays } from 'date-fns';

const formatTimeseriesData = (clusterData) => {
    if (!clusterData) {
        return null;
    }
    // set a baseline zero'd object for the past week
    const baselineData = {};
    const xAxisBuckets = [];
    for (let i = 6; i >= 0; i -= 1) {
        const key = format(subDays(new Date(), i), 'MMM DD');
        baselineData[key] = 0;
        xAxisBuckets.push(key);
    }
    // set severities in timeAlertMap to have this zero'd data
    const timeAlertMap = {};
    const timeAlertInitialMap = {}; // this is the number of initial alerts that have come before
    Object.keys(severityColorMap).forEach((severity) => {
        timeAlertMap[severity] = cloneDeep(baselineData);
        timeAlertInitialMap[severity] = 0;
    });

    // populate actual data into timeAlertMap
    clusterData.severities.forEach((severityObj) => {
        const { severity, events } = severityObj;
        events.forEach((alert) => {
            const time = format(parseInt(alert.time, 10), 'MMM DD');
            const alerts = timeAlertMap[severity][time];
            if (alerts !== undefined) {
                switch (alert.type) {
                    case 'CREATED':
                        timeAlertMap[severity][time] += 1;
                        break;
                    case 'REMOVED':
                        timeAlertMap[severity][time] -= 1;
                        break;
                    default:
                        break;
                }
            } else {
                timeAlertInitialMap[severity] += 1;
            }
        });
    });

    Object.keys(severityColorMap).forEach((severity) => {
        let runningSum = timeAlertInitialMap[severity];
        Object.keys(baselineData).forEach((time) => {
            const prevVal = timeAlertMap[severity][time];
            timeAlertMap[severity][time] += runningSum;
            runningSum += prevVal;
        });
    });

    // set data format for line chart
    const cluster = {};
    cluster.data = Object.keys(baselineData).map((time) => ({
        time,
        low: timeAlertMap.LOW_SEVERITY[time],
        medium: timeAlertMap.MEDIUM_SEVERITY[time],
        high: timeAlertMap.HIGH_SEVERITY[time],
        critical: timeAlertMap.CRITICAL_SEVERITY[time],
    }));
    cluster.name = clusterData.cluster;

    return cluster;
};

const AlertsByTimeseriesChart = ({ alertsByTimeseries }) => {
    if (!alertsByTimeseries || !alertsByTimeseries.length) {
        return (
            <NoResultsMessage message="No data available. Please ensure your cluster is properly configured." />
        );
    }
    return (
        <div className="p-0 h-64 w-full overflow-hidden">
            <Slider {...slickSettings}>
                {alertsByTimeseries.map((cluster) => {
                    const { data, name } = formatTimeseriesData(cluster);
                    return (
                        <div className="h-64" key={name}>
                            <CustomLineChart
                                data={data}
                                name={name}
                                xAxisDataKey="time"
                                yAxisDataKey=""
                            >
                                <Line
                                    type="monotone"
                                    dataKey="low"
                                    stroke={severityColorMap.LOW_SEVERITY}
                                />
                                <Line
                                    type="monotone"
                                    dataKey="medium"
                                    stroke={severityColorMap.MEDIUM_SEVERITY}
                                />
                                <Line
                                    type="monotone"
                                    dataKey="high"
                                    stroke={severityColorMap.HIGH_SEVERITY}
                                />
                                <Line
                                    type="monotone"
                                    dataKey="critical"
                                    stroke={severityColorMap.CRITICAL_SEVERITY}
                                />
                            </CustomLineChart>
                        </div>
                    );
                })}
            </Slider>
        </div>
    );
};

AlertsByTimeseriesChart.propTypes = {
    alertsByTimeseries: PropTypes.arrayOf(
        PropTypes.shape({
            cluster: PropTypes.string.isRequired,
            severities: PropTypes.arrayOf(
                PropTypes.shape({
                    severity: PropTypes.string.isRequired,
                    events: PropTypes.arrayOf(
                        PropTypes.shape({
                            id: PropTypes.string.isRequired,
                            time: PropTypes.string.isRequired,
                            type: PropTypes.string.isRequired,
                        })
                    ),
                })
            ),
        })
    ),
};

AlertsByTimeseriesChart.defaultProps = {
    alertsByTimeseries: [],
};

export default AlertsByTimeseriesChart;
