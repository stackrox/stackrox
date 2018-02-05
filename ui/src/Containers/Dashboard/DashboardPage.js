import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { Line, BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import TwoLevelPieChart from 'Components/visuals/TwoLevelPieChart';
import CustomLineChart from 'Components/visuals/CustomLineChart';
import DashboardBenchmarks from 'Containers/Dashboard/DashboardBenchmarks';
import SeverityTile from 'Containers/Dashboard/SeverityTile';

import axios from 'axios';
import { format, subDays } from 'date-fns';
import * as Icon from 'react-feather';
import { severityLabels } from 'messages/common';

//  @TODO: Have one source of truth for severity colors
const severityColorMap = {
    CRITICAL_SEVERITY: 'hsl(7, 100%, 55%)',
    HIGH_SEVERITY: 'hsl(349, 100%, 78%)',
    MEDIUM_SEVERITY: 'hsl(20, 100%, 78%)',
    LOW_SEVERITY: 'hsl(42, 100%, 84%)'
};

const policyCategoriesMap = {
    CONTAINER_CONFIGURATION: {
        label: 'Container Configuration',
        icon: <Icon.Grid className="h-4 w-4 mr-3" />
    },
    IMAGE_ASSURANCE: {
        label: 'Image Assurance',
        icon: <Icon.Copy className="h-4 w-4 mr-3" />
    },
    PRIVILEGES_CAPABILITIES: {
        label: 'Privileges and Capabilities',
        icon: <Icon.Lock className="h-4 w-4 mr-3" />
    }
};

const benchmarkNames = ['CIS Benchmark', 'Swarm Benchmark', 'Kubernetes v1.2.0 Benchmark'];

const reducer = (action, prevState, nextState) => {
    switch (action) {
        case 'UPDATE_VIOLATIONS_BY_POLICY_CATEGORY':
            return { violatonsByPolicyCategory: nextState.violatonsByPolicyCategory };
        case 'UPDATE_VIOLATIONS_BY_CLUSTERS':
            return { violationsByCluster: nextState.violationsByCluster };
        case 'UPDATE_EVENTS_BY_TIME':
            return { eventsByTime: nextState.eventsByTime };
        case 'UPDATE_BENCHMARKS':
            return { benchmarks: nextState.benchmarks };
        default:
            return prevState;
    }
};

class DashboardPage extends Component {
    static propTypes = {
        history: PropTypes.shape({
            push: PropTypes.func.isRequired
        }).isRequired
    }

    constructor(props) {
        super(props);

        this.colorBy = d => d.color;

        this.state = {
            violatonsByPolicyCategory: [],
            violationsByCluster: [],
            benchmarks: {},
            eventsByTime: []
        };
    }

    componentDidMount() {
        this.getViolationsByPolicyCategory();
        this.getViolationsByCluster();
        this.getEventsByTime();
        this.getBenchmarks();
    }

    getViolationsByPolicyCategory = () => axios.get('/v1/alerts/counts', {
        params: { group_by: 'CATEGORY', 'request.stale': false }
    }).then((response) => {
        const violatonsByPolicyCategory = response.data.groups;
        if (!violatonsByPolicyCategory) return;
        this.update('UPDATE_VIOLATIONS_BY_POLICY_CATEGORY', { violatonsByPolicyCategory });
    }).catch((error) => {
        console.error(error);
    });

    getViolationsByCluster = () => axios.get('/v1/alerts/counts', {
        params: { group_by: 'CLUSTER', 'request.stale': false }
    }).then((response) => {
        const violationsByCluster = response.data.groups;
        if (!violationsByCluster) return;
        this.update('UPDATE_VIOLATIONS_BY_CLUSTERS', { violationsByCluster });
    }).catch((error) => {
        console.error(error);
    });

    getEventsByTime = () => axios.get('/v1/alerts/timeseries').then((response) => {
        const eventsByTime = response.data.events;
        if (!eventsByTime) return;
        this.update('UPDATE_EVENTS_BY_TIME', { eventsByTime });
    }).catch((error) => {
        console.error(error);
    });

    getBenchmarks = () => {
        const promise = new Promise((resolve) => {
            const promises = [];
            benchmarkNames.forEach((benchmarkName) => {
                promises.push(axios.get(`/v1/benchmarks/results/grouped/${benchmarkName}`));
            });
            Promise.all(promises).then((values) => {
                const { benchmarks } = this.state;
                values.forEach((value, i) => {
                    benchmarks[benchmarkNames[i]] = value.data.benchmarks;
                });
                this.update('UPDATE_BENCHMARKS', { benchmarks });
                resolve(values);
            });
        });
        return promise;
    };

    makeBarClickHandler = (cluster, severity) => () => {
        this.props.history.push(`/violations?severity=${severity}&cluster=${cluster}`);
    }

    update = (action, nextState) => {
        this.setState(prevState => reducer(action, prevState, nextState));
    }

    renderEventsByTime = () => {
        if (!this.state.eventsByTime) return '';
        const timeEventMap = {};
        for (let i = 6; i >= 0; i -= 1) {
            timeEventMap[format(subDays(new Date(), i), 'MMM DD')] = 0;
        }
        this.state.eventsByTime.forEach((event) => {
            const time = format(parseInt(event.time, 10), 'MMM DD');
            const events = timeEventMap[time];
            if (events !== undefined) {
                switch (event.type) {
                    case 'CREATED':
                        timeEventMap[time] += 1;
                        break;
                    case 'REMOVED':
                        timeEventMap[time] -= 1;
                        break;
                    default:
                        break;
                }
            }
        });
        const data = Object.keys(timeEventMap).map(time => ({ time, events: timeEventMap[time] }));
        return (
            <CustomLineChart data={data} xAxisDataKey="time" yAxisDataKey="Events">
                <Line type="monotone" dataKey="events" stroke="#82ca9d" />
            </CustomLineChart>
        );
    }

    renderViolationsByCluster = () => {
        if (!this.state.violationsByCluster) return '';
        const data = this.state.violationsByCluster.map((cluster) => {
            const dataPoint = {
                name: cluster.group,
                Critical: 0,
                High: 0,
                Medium: 0,
                Low: 0
            };
            cluster.counts.forEach((d) => {
                dataPoint[severityLabels[d.severity]] = parseInt(d.count, 10);
            });
            return dataPoint;
        });
        return (
            <ResponsiveContainer>
                <BarChart
                    data={data}
                    margin={
                        {
                            top: 5,
                            right: 30,
                            left: 20,
                            bottom: 5
                        }
                    }
                >
                    <XAxis dataKey="name" />
                    <YAxis
                        domain={[0, 'dataMax']}
                        allowDecimals={false}
                        label={{
                            value: 'Count', angle: -90, position: 'insideLeft', textAnchor: 'middle'
                        }}
                    />
                    <CartesianGrid strokeDasharray="3 3" />
                    <Tooltip />
                    <Legend horizontalAlign="right" wrapperStyle={{ lineHeight: '40px' }} />
                    {
                        Object.keys(severityLabels).map((severity) => {
                            const arr = [];
                            const bar = (
                                <Bar
                                    name={severityLabels[severity]}
                                    key={severityLabels[severity]}
                                    dataKey={severityLabels[severity]}
                                    fill={severityColorMap[severity]}
                                >
                                    {
                                        data.map(entry => (
                                            <Cell
                                                key={entry.name}
                                                className="cursor-pointer"
                                                onClick={
                                                    this.makeBarClickHandler(entry.name, severity)
                                                }
                                            />
                                        ))
                                    }
                                </Bar>
                            );
                            arr.push(bar);
                            return arr;
                        })
                    }
                </BarChart>
            </ResponsiveContainer>
        );
    }

    renderViolationsByPolicyCategory = () => {
        if (!this.state.violatonsByPolicyCategory) return '';
        return this.state.violatonsByPolicyCategory.map((policyType) => {
            const data = policyType.counts.map(d => ({
                name: severityLabels[d.severity],
                value: parseInt(d.count, 10),
                color: severityColorMap[d.severity],
                onClick: () => {
                    this.props.history.push(`/violations?category=${policyType.group}&severity=${d.severity}`);
                }
            }));
            return (
                <div className="p-8 w-full lg:w-1/2" key={policyType.group}>
                    <div className="flex flex-col p-4 bg-white rounded-sm shadow">
                        <h2 className="flex items-center text-lg text-base font-sans text-base-600 py-4 tracking-wide">
                            {policyCategoriesMap[policyType.group].icon}
                            {policyCategoriesMap[policyType.group].label}
                        </h2>
                        <div className="flex flex-1 m-4 h-64">
                            <TwoLevelPieChart data={data} />
                        </div>
                    </div>
                </div>
            );
        });
    };

    renderEnvironmentRisk = () => {
        const counts = {
            CRITICAL_SEVERITY: 0,
            HIGH_SEVERITY: 0,
            MEDIUM_SEVERITY: 0,
            LOW_SEVERITY: 0
        };
        this.state.violationsByCluster.forEach((cluster) => {
            cluster.counts.forEach((d) => {
                const count = parseInt(d.count, 10);
                counts[d.severity] += count;
            });
        });
        const severities = Object.keys(counts);
        return (
            <div className="flex flex-1 flex-col w-full">
                <h2 className="flex items-center text-xl text-base font-sans text-base-600 pb-8 tracking-wide font-500">Environment Risk</h2>
                <div className="flex">
                    {
                        severities.map((severity, i) => (
                            <SeverityTile
                                severity={severity}
                                count={counts[severity]}
                                color={severityColorMap[severity]}
                                index={i}
                                key={severity}
                            />
                        ))
                    }
                </div>
            </div>
        );
    };

    renderBenchmarks = () => (
        <div className="flex flex-1 flex-col w-full">
            <h2 className="flex items-center text-xl text-base font-sans text-base-600 pb-8 tracking-wide font-500">Benchmarks</h2>
            <DashboardBenchmarks benchmarks={this.state.benchmarks} />
        </div>
    );

    render() {
        return (
            <section className="w-full h-full transition">
                <div className="flex bg-white border-b border-primary-500">
                    <div className="flex flex-1 flex-col w-1/2 p-8">
                        {this.renderEnvironmentRisk()}
                    </div>
                    <div className="flex flex-1 flex-col w-1/2 p-8 border-l border-primary-200">
                        {this.renderBenchmarks()}
                    </div>
                </div>
                <div className="overflow-auto">
                    <div className="flex flex-col w-full">
                        <div className="flex w-full flex-wrap">
                            <div className="p-8 md:w-full lg:w-1/2">
                                <div className="flex flex-col p-4 bg-white rounded-sm shadow">
                                    <h2 className="flex items-center text-lg text-base font-sans text-base-600 py-4 tracking-wide">
                                        <Icon.Layers className="h-4 w-4 mr-3" />
                                        Violations by Cluster
                                    </h2>
                                    <div className="flex flex-1 m-4 h-64">{this.renderViolationsByCluster()}</div>
                                </div>
                            </div>
                            <div className="p-8 md:w-full lg:w-1/2">
                                <div className="flex flex-col p-4 bg-white rounded-sm shadow">
                                    <h2 className="flex items-center text-lg text-base font-sans text-base-600 py-4 tracking-wide">
                                        <Icon.AlertTriangle className="h-4 w-4 mr-3" />
                                        Events by Time
                                    </h2>
                                    <div className="flex flex-1 m-4 h-64">{this.renderEventsByTime()}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div className="flex flex-col w-full">
                        <div className="flex w-full flex-wrap">
                            {this.renderViolationsByPolicyCategory()}
                        </div>
                    </div>
                </div>
            </section>
        );
    }
}

export default DashboardPage;
