import React from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import { createStructuredSelector } from 'reselect';
import { connect } from 'react-redux';

import { selectors } from 'reducers';
import { addFieldArrayHandler, removeFieldArrayHandler } from '../utils';
import SingleScope from './SingleScope';

const ScopeArray = ({ fields, clusters, deployments, isDeploymentScope }) => {
    const clusterOptions = [{ label: 'Cluster', value: '' }].concat(
        clusters.map((cluster) => ({
            label: cluster.name,
            value: cluster.id,
        }))
    );

    const deploymentOptions = isDeploymentScope
        ? [{ label: 'Deployment', value: '' }].concat(
              deployments.map(({ deployment }) => ({
                  label: deployment.name,
                  value: deployment.name,
              }))
          )
        : [];
    return (
        <div className="w-full">
            <div className="w-full text-right">
                <button
                    className="text-base-500"
                    onClick={addFieldArrayHandler(fields, {})}
                    type="button"
                    data-testid="add-scope"
                >
                    <Icon.PlusSquare size="40" />
                </button>
            </div>
            {fields.map((pair, index) => (
                <SingleScope
                    key={pair}
                    deleteScopeHandler={removeFieldArrayHandler(fields, index)}
                    isDeploymentScope={isDeploymentScope}
                    clusterOptions={clusterOptions}
                    deploymentOptions={deploymentOptions}
                    fieldBasePath={pair}
                />
            ))}
        </div>
    );
};

ScopeArray.propTypes = {
    fields: PropTypes.shape({
        map: PropTypes.func.isRequired,
    }).isRequired,
    clusters: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    deployments: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    isDeploymentScope: PropTypes.bool.isRequired,
};

const mapStateToProps = createStructuredSelector({
    clusters: selectors.getClusters,
    deployments: selectors.getDeployments,
});

export default connect(mapStateToProps, {})(ScopeArray);
