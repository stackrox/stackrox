import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createSelector, createStructuredSelector } from 'reselect';

import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import BenchmarksPage from 'Containers/Compliance/BenchmarksPage';
import PageHeader from 'Components/PageHeader';

const CompliancePage = props => (
    <section className="flex flex-1 h-full">
        <div className="flex flex-1 flex-col">
            <PageHeader header={`${props.cluster ? props.cluster.name : ''}`} subHeader="Cluster" />
            <div className="flex flex-1">
                <Tabs className="bg-base-100" headers={props.benchmarkTabs}>
                    {props.benchmarkTabs.map(benchmark => (
                        <TabContent key={benchmark.benchmarkName}>
                            <BenchmarksPage
                                benchmarkName={benchmark.benchmarkName}
                                benchmarkId={benchmark.benchmarkId}
                            />
                        </TabContent>
                    ))}
                </Tabs>
            </div>
        </div>
    </section>
);

CompliancePage.propTypes = {
    benchmarkTabs: PropTypes.arrayOf(
        PropTypes.shape({
            benchmarkName: PropTypes.string,
            text: PropTypes.string,
            disabled: PropTypes.bool
        })
    ).isRequired,
    cluster: PropTypes.shape({
        name: PropTypes.string.isRequired,
        id: PropTypes.string.isRequired
    })
};

CompliancePage.defaultProps = {
    cluster: null // cluster data is being loaded
};

const getClusterId = (state, props) => props.match.params.clusterId;

const getBenchmarkTabs = createSelector(
    [selectors.getBenchmarks, selectors.getClusters, getClusterId],
    (benchmarks, clusters, clusterId) => {
        let selectedCluster = clusters.find(obj => obj.id === clusterId);
        if (!selectedCluster) [selectedCluster] = clusters;
        const result = benchmarks
            .map(benchmark => {
                const available = benchmark.clusterTypes.reduce(
                    (val, type) => val || (selectedCluster && selectedCluster.type === type),
                    false
                );

                return {
                    benchmarkName: benchmark.name,
                    benchmarkId: benchmark.id,
                    text: benchmark.name,
                    disabled: !available
                };
            })
            .sort((a, b) => (a.disabled < b.disabled ? -1 : a.disabled > b.disabled));
        return result;
    }
);

const getCluster = createSelector([selectors.getClusters, getClusterId], (clusters, clusterId) => {
    let selectedCluster = clusters.find(obj => obj.id === clusterId);
    if (!selectedCluster) [selectedCluster] = clusters;
    return selectedCluster;
});

const mapStateToProps = createStructuredSelector({
    benchmarkTabs: getBenchmarkTabs,
    cluster: getCluster
});

export default connect(mapStateToProps)(CompliancePage);
