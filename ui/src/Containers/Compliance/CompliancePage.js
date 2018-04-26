import React from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
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
            <PageHeader header={props.clusterName} />
            <div className="flex flex-1">
                <Tabs className="bg-white" headers={props.benchmarkTabs}>
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
    clusterName: PropTypes.string.isRequired
};

CompliancePage.defaultPropTypes = {
    location: ReactRouterPropTypes.location.required
};

const getBenchmarkTabs = createSelector([selectors.getBenchmarks], benchmarks =>
    benchmarks
        .map(benchmark => ({
            benchmarkName: benchmark.name,
            benchmarkId: benchmark.id,
            text: benchmark.name,
            disabled: !benchmark.available
        }))
        .sort((a, b) => (a.disabled < b.disabled ? -1 : a.disabled > b.disabled))
);

const getClusterName = createSelector(
    [selectors.getClusters, selectors.getLocation],
    (clusters, location) => {
        const id = location.pathname.split('/').pop();
        const cluster = clusters.find(obj => obj.id === id);
        if (cluster) return cluster.name;
        return '';
    }
);

const mapStateToProps = createStructuredSelector({
    benchmarkTabs: getBenchmarkTabs,
    clusterName: getClusterName
});

export default connect(mapStateToProps)(CompliancePage);
