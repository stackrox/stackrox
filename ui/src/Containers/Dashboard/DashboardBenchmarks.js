import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

const benchmarkResultsMap = {
    PASS: 'hsl(225, 95%, 70%)',
    WARN: 'hsl(257, 61%, 71%)',
    INFO: 'hsl(293, 64%, 76%)',
    NOTE: 'hsl(210, 80%, 73%)'
};

class DashboardBenchmarks extends Component {
    static propTypes = {
        cluster: PropTypes.shape({
            benchmarks: PropTypes.arrayOf(PropTypes.shape()).isRequired,
            clusterId: PropTypes.string.isRequired,
            clusterName: PropTypes.string.isRequired
        }).isRequired
    };

    hasBenchmarks = () => {
        let doesHaveBenchmarks = false;
        this.props.cluster.benchmarks.forEach(benchmark => {
            if (benchmark.counts.length) {
                doesHaveBenchmarks = true;
            }
        });
        return doesHaveBenchmarks;
    };

    renderBenchmarks = () =>
        this.props.cluster.benchmarks.map(benchmark => {
            if (!benchmark.counts.length) return '';
            const results = {
                PASS: 0,
                WARN: 0,
                INFO: 0,
                NOTE: 0
            };
            let total = 0;
            benchmark.counts.forEach(count => {
                const value = parseInt(count.count, 10);
                results[count.status] += value;
                total += value;
            });
            return (
                <div className="pb-3 flex w-full items-center" key={benchmark.benchmark}>
                    <Link
                        className="text-sm text-primary-700 hover:text-primary-800 tracking-wide underline w-1/3 text-left"
                        to={`/main/compliance/${this.props.cluster.clusterId}`}
                    >
                        {benchmark.benchmark}
                    </Link>

                    <div className="flex flex-1 w-1/2 h-2">
                        {Object.keys(results).map(result => {
                            const width = Math.ceil((results[result] / total) * 100);
                            if (!width) return '';
                            const backgroundStyle = {
                                backgroundColor: benchmarkResultsMap[result],
                                width: `${width}%`
                            };
                            return (
                                <div
                                    className="border-r border-base-100"
                                    style={backgroundStyle}
                                    key={result}
                                />
                            );
                        })}
                    </div>
                </div>
            );
        });

    renderLegend = () =>
        Object.keys(benchmarkResultsMap).map(result => {
            const backgroundStyle = {
                backgroundColor: benchmarkResultsMap[result]
            };
            return (
                <div className="flex items-center" key={result}>
                    <div className="h-1 w-8 mr-4" style={backgroundStyle} />
                    <div className="text-sm text-primary-800 tracking-wide capitalize">
                        {result}
                    </div>
                </div>
            );
        });

    render() {
        if (!this.hasBenchmarks()) {
            return (
                <div className="h-full">
                    <h2 className="bg-base-100 inline-block leading-normal mb-4 p-3 pb-2 pl-6 pr-4 rounded-r-full text-base-600 text-lg text-primary-800 tracking-wide tracking-widest uppercase">
                        {`Benchmarks for "${this.props.cluster.clusterName}"`}
                    </h2>
                    <div className="flex flex-col text-center font-700 items-center px-6">
                        <div className="flex flex-col p-4">
                            <span className="mb-4"> No Benchmark Results available.</span>

                            <Link
                                to={`/main/compliance/${this.props.cluster.clusterId}`}
                                className="no-underline"
                            >
                                <button
                                    type="button"
                                    className="bg-primary-600 px-5 py-3 text-base-100 font-600 rounded-sm uppercase text-sm hover:bg-primary-700"
                                >
                                    Scan your cluster
                                </button>
                            </Link>
                        </div>
                    </div>
                </div>
            );
        }
        return (
            <div>
                <h2 className="bg-base-100 inline-block leading-normal mb-4 p-3 pb-2 pl-6 pr-4 rounded-r-full text-base-600 text-lg text-primary-800 tracking-wide tracking-widest uppercase">
                    {`Benchmarks for "${this.props.cluster.clusterName}"`}
                </h2>
                <div className="pt-4 px-6">{this.renderBenchmarks()}</div>
                <div className="flex flex-1 w-full pt-4 justify-between px-6">
                    {this.renderLegend()}
                </div>
            </div>
        );
    }
}

export default DashboardBenchmarks;
