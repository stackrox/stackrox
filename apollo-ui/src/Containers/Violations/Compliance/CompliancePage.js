import React from 'react';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';

import BenchmarksPage from 'Containers/Violations/Compliance/BenchmarksPage';

const CompliancePage = () => {
    const headers = [{ text: 'CIS Docker Benchmark', disabled: false }, { text: 'Swarm Benchmark', disabled: false }, { text: 'Kubernetes Benchmark', disabled: true }];
    return (
        <div className="flex flex-1">
            <Tabs className="bg-white" headers={headers}>
                <TabContent>
                    <BenchmarksPage benchmarksResults="CIS Benchmark" benchmarksTrigger="CIS Benchmark" />
                </TabContent>
                <TabContent>
                    <BenchmarksPage benchmarksResults="Swarm Benchmark" benchmarksTrigger="Swarm Benchmark" />
                </TabContent>
                <TabContent>
                    <BenchmarksPage benchmarksResults="Kubernetes Benchmark" benchmarksTrigger="Kubernetes Benchmark" />
                </TabContent>
            </Tabs>
        </div>
    );
};

export default CompliancePage;
