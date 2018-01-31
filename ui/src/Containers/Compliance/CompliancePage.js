import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';

import BenchmarksPage from 'Containers/Compliance/BenchmarksPage';
import axios from 'axios';

class CompliancePage extends Component {
    constructor(props) {
        super(props);

        this.state = {
            benchmarkTabs: [],
        };
    }

    componentDidMount() {
        this.getBenchmarks();
    }

    getBenchmarks() {
        axios.get('/v1/clusters').then((clusterResponse) => {
            const { clusters } = clusterResponse.data;
            const clusterTypes = new Set(clusters.map(c => c.type));

            return axios.get('/v1/benchmarks/configs').then((configResponse) => {
                const { benchmarks } = configResponse.data;
                const benchmarkTabs = benchmarks.map((benchmark) => {
                    const enabled = benchmark.clusterTypes.reduce((val, type) =>
                        val || clusterTypes.has(type), false);
                    return {
                        benchmarkName: benchmark.name,
                        text: benchmark.name,
                        disabled: !enabled
                    };
                }).sort((a, b) => (a.disabled < b.disabled ? -1 : a.disabled > b.disabled));
                this.setState({ benchmarkTabs });
            });
        }).catch((error) => {
            console.error(error);
        });
    }

    render() {
        return (
            <section className="flex flex-1 h-full">
                <div className="flex flex-1">
                    <Tabs className="bg-white" headers={this.state.benchmarkTabs}>
                        {
                            this.state.benchmarkTabs.map(benchmark => (
                                <TabContent key={benchmark.benchmarkName}>
                                    <BenchmarksPage benchmarkName={benchmark.benchmarkName} />
                                </TabContent>
                            ))
                        }
                    </Tabs>
                </div>
            </section>
        );
    }
}

export default CompliancePage;
