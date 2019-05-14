import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import { actions as globalSearchActions } from 'reducers/globalSearch';

import 'rc-tooltip/assets/bootstrap.css';

const titleMap = {
    numClusters: { singular: 'Cluster', plural: 'Clusters' },
    numNodes: { singular: 'Node', plural: 'Nodes' },
    numAlerts: { singular: 'Violation', plural: 'Violations' },
    numDeployments: { singular: 'Deployment', plural: 'Deployments' },
    numImages: { singular: 'Image', plural: 'Images' },
    numSecrets: { singular: 'Secret', plural: 'Secrets' }
};

const SummaryCounts = ({ counts }) => {
    if (!counts) return '';
    return (
        <ul className="flex uppercase text-sm p-0 w-full">
            {Object.entries(titleMap).map(([key, titles]) => (
                <li
                    key={key}
                    className="flex flex-col border-r border-base-400 border-dashed px-3 lg:w-24 md:w-20 no-underline py-3 text-base-500 items-center justify-center font-condensed"
                >
                    <div className="text-3xl tracking-widest">{counts[key]}</div>
                    <div className="text-sm pt-1 tracking-wide">
                        {counts[key] === '1' ? titles.singular : titles.plural}
                    </div>
                </li>
            ))}
        </ul>
    );
};

SummaryCounts.propTypes = {
    counts: PropTypes.shape({
        numClusters: PropTypes.string,
        numNodes: PropTypes.string,
        numAlerts: PropTypes.string,
        numDeployments: PropTypes.string,
        numImages: PropTypes.string,
        numSecrets: PropTypes.string
    })
};

SummaryCounts.defaultProps = {
    counts: null
};

const mapStateToProps = createStructuredSelector({
    counts: selectors.getSummaryCounts
});

const mapDispatchToProps = dispatch => ({
    toggleGlobalSearchView: () => dispatch(globalSearchActions.toggleGlobalSearchView())
});

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(SummaryCounts)
);
