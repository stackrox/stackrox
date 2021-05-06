import React, { useMemo } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useParams } from 'react-router-dom';

import { selectors } from 'reducers';
import useNetworkBaselineSimulation from 'Containers/Network/useNetworkBaselineSimulation';

import Tab from 'Components/Tab';
import NetworkEntityTabbedOverlay from 'Components/NetworkEntityTabbedOverlay';
import DeploymentDetails from './DeploymentDetails';
import NetworkPoliciesDetail from './NetworkPoliciesDetail';
import Flows from './Flows';
import BaselineSimulation from './BaselineSimulation';

function NetworkDeploymentOverlay({
    selectedDeployment,
    filterState,
    lastUpdatedTimestamp,
    networkNodeMap,
}) {
    const { isBaselineSimulationOn } = useNetworkBaselineSimulation();
    const { deploymentId } = useParams();

    const entityIdToNamespaceMap = useMemo(() => {
        return Object.keys(networkNodeMap).reduce((accumulator, entityId) => {
            const val = networkNodeMap[entityId];
            const entity = val?.active?.entity || val?.allowed?.entity;
            if (entity.type === 'DEPLOYMENT') {
                accumulator[entityId] = entity.deployment.namespace;
            }
            return accumulator;
        }, {});
    }, [networkNodeMap]);

    return (
        <NetworkEntityTabbedOverlay
            entityName={selectedDeployment.name}
            entityType={selectedDeployment.type}
        >
            <Tab title="Flows">
                {isBaselineSimulationOn ? (
                    <BaselineSimulation deploymentId={deploymentId} />
                ) : (
                    <Flows
                        selectedDeployment={selectedDeployment}
                        deploymentId={deploymentId}
                        filterState={filterState}
                        lastUpdatedTimestamp={lastUpdatedTimestamp}
                        entityIdToNamespaceMap={entityIdToNamespaceMap}
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
    networkNodeMap: PropTypes.shape({}).isRequired,
    filterState: PropTypes.number.isRequired,
    lastUpdatedTimestamp: PropTypes.instanceOf(Date).isRequired,
};

const mapStateToProps = createStructuredSelector({
    selectedDeployment: selectors.getSelectedNode,
    filterState: selectors.getNetworkGraphFilterMode,
    networkNodeMap: selectors.getNetworkNodeMap,
    lastUpdatedTimestamp: selectors.getLastUpdatedTimestamp,
});

export default connect(mapStateToProps, null)(NetworkDeploymentOverlay);
