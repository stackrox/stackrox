import React, { Component } from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

const benchmarkResultsMap = {
    PASS: 'hsl(223, 95%, 70%)',
    WARN: 'hsl(245, 61%, 71%)',
    INFO: 'hsl(297, 64%, 76%)',
    NOTE: 'hsl(204, 80%, 73%)'
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
                <div className="pb-3 flex w-full" key={benchmark.benchmark}>
                    <Link
                        className="text-sm text-primary-500 tracking-wide underline w-1/3 text-left"
                        to={`/main/compliance/${this.props.cluster.clusterId}`}
                    >
                        {benchmark.benchmark}
                    </Link>
                    <div className="flex flex-1 w-2/3 h-2">
                        {Object.keys(results).map(result => {
                            const width = Math.ceil(results[result] / total * 100);
                            if (!width) return '';
                            const backgroundStyle = {
                                backgroundColor: benchmarkResultsMap[result],
                                width: `${width}%`
                            };
                            return (
                                <div
                                    className="border-r border-white"
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
                <div className="flex flex-1 w-full justify-center items-center" key={result}>
                    <div className="h-1 w-6 mr-4" style={backgroundStyle} />
                    <div className="text-sm text-base-600 tracking-wide capitalize">{result}</div>
                </div>
            );
        });

    render() {
        if (!this.hasBenchmarks()) {
            return (
                <div className="h-full">
                    <h2 className="flex text-xl text-base font-sans text-base-600 tracking-wide font-500 capitalize">
                        {this.props.cluster.clusterName} Benchmarks
                    </h2>
                    <div className="flex flex-1 items-center justify-center h-full">
                        No Benchmark Results
                    </div>
                </div>
            );
        }
        return (
            <div>
                <h2 className="flex text-xl text-base font-sans text-base-600 pb-4 tracking-wide font-500 capitalize">
                    {this.props.cluster.clusterName} Benchmarks
                </h2>
                <div className="pt-4">{this.renderBenchmarks()}</div>
                <div className="flex flex-1 w-full pt-4">{this.renderLegend()}</div>
            </div>
        );
    }
}

export default DashboardBenchmarks;
