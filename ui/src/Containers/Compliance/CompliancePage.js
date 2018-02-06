import React, { Component } from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import BenchmarksPage from 'Containers/Compliance/BenchmarksPage';
import retrieveBenchmarks from 'Providers/BenchmarksService';

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
        retrieveBenchmarks().then((benchmarks) => {
            const benchmarkTabs = benchmarks.map(benchmark => ({
                benchmarkName: benchmark.name,
                text: benchmark.name,
                disabled: !benchmark.available
            })).sort((a, b) => (a.disabled < b.disabled ? -1 : a.disabled > b.disabled));
            this.setState({ benchmarkTabs });
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
