import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useParams } from 'react-router-dom';

import { selectors } from 'reducers';
import { useNetworkBaselineSimulation } from 'Containers/Network/baselineSimulationContext';

import Tab from 'Components/Tab';
import NetworkEntityTabbedOverlay from 'Components/NetworkEntityTabbedOverlay';
import DeploymentDetails from './DeploymentDetails';
import NetworkPoliciesDetail from './NetworkPoliciesDetail';
import Flows from './Flows';

function NetworkDeploymentOverlay({ selectedDeployment, filterState, lastUpdatedTimestamp }) {
    const { isBaselineSimulationOn } = useNetworkBaselineSimulation();
    const { deploymentId } = useParams();

    return (
        <NetworkEntityTabbedOverlay
            entityName={selectedDeployment.name}
            entityType={selectedDeployment.type}
        >
            <Tab title="Flows">
                {isBaselineSimulationOn ? null : (
                    <Flows
                        selectedDeployment={selectedDeployment}
                        deploymentId={deploymentId}
                        filterState={filterState}
                        lastUpdatedTimestamp={lastUpdatedTimestamp}
                    />
                )}
            </Tab>
            <Tab title="Network Policies">
                <NetworkPoliciesDetail policyIds={selectedDeployment.policyIds} />
            </Tab>
            <Tab title="Details">
                <DeploymentDetails deploymentId={deploymentId} />
            </Tab>
        </NetworkEntityTabbedOverlay>
    );
}

NetworkDeploymentOverlay.propTypes = {
    selectedDeployment: PropTypes.shape({
        id: PropTypes.string.isRequired,
        name: PropTypes.string.isRequired,
        type: PropTypes.string.isRequired,
        edges: PropTypes.arrayOf(PropTypes.shape({})),
        policyIds: PropTypes.arrayOf(PropTypes.string),
    }).isRequired,
    filterState: PropTypes.number.isRequired,
    lastUpdatedTimestamp: PropTypes.instanceOf(Date).isRequired,
};

const mapStateToProps = createStructuredSelector({
    selectedDeployment: selectors.getSelectedNode,
    filterState: selectors.getNetworkGraphFilterMode,
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
});

export default connect(mapStateToProps, null)(NetworkDeploymentOverlay);
