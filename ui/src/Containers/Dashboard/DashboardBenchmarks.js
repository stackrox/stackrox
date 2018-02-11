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
        benchmarks: PropTypes.shape({}).isRequired
    };

    hasBenchmarks = () => {
        let doesHaveBenchmarks = false;
        Object.keys(this.props.benchmarks).forEach(benchmarkName => {
            if (this.props.benchmarks[benchmarkName].length) {
                doesHaveBenchmarks = true;
            }
        });
        return doesHaveBenchmarks;
    };

    renderBenchmarks = () =>
        Object.keys(this.props.benchmarks).map(benchmarkName => {
            const benchmarks = this.props.benchmarks[benchmarkName];
            if (!benchmarks.length) return '';
            const results = {
                PASS: 0,
                WARN: 0,
                INFO: 0,
                NOTE: 0
            };
            let total = 0;
            benchmarks[0].checks.forEach(result => {
                Object.keys(result.aggregatedResults).forEach(aggregatedResult => {
                    if (results[aggregatedResult] !== undefined) {
                        const value = parseInt(result.aggregatedResults[aggregatedResult], 10);
                        results[aggregatedResult] += value;
                        total += value;
                    }
                });
            });
            return (
                <div className="pb-3 flex w-full" key={benchmarkName}>
                    <Link
                        className="text-sm text-primary-500 tracking-wide underline w-1/3 text-left"
                        to="/compliance"
                    >
                        {benchmarkName}
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
                <div className="flex flex-1 items-center justify-center">No Benchmark Results</div>
            );
        }
        return (
            <div>
                <div>{this.renderBenchmarks()}</div>
                <div className="flex flex-1 w-full pt-4">{this.renderLegend()}</div>
            </div>
        );
    }
}

export default DashboardBenchmarks;
