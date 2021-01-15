import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useParams } from 'react-router-dom';

import { selectors } from 'reducers';
import { nodeTypes } from 'constants/networkGraph';
import useNavigateToEntity from 'hooks/useNavigateToEntity';

import Tab from 'Components/Tab';
import NetworkEntityTabbedOverlay from 'Components/NetworkEntityTabbedOverlay';
import BinderTabs from 'Components/BinderTabs';
import NetworkFlows from './NetworkFlows';
import BaselineSettings from './BaselineSettings';
import DeploymentDetails from './DeploymentDetails';
import NetworkPoliciesDetail from './NetworkPoliciesDetail';

function getDeploymentEdges(deployment) {
    const edges = deployment.edges.filter(
        ({ data: { destNodeName, destNodeNamespace, source, target, destNodeType } }) =>
            destNodeNamespace &&
            destNodeName &&
            (source !== target || destNodeType !== nodeTypes.DEPLOYMENT)
    );
    return edges;
}

function NetworkDeploymentOverlay({ selectedDeployment, filterState, lastUpdatedTimestamp }) {
    const onNavigateToEntity = useNavigateToEntity();
    const { deploymentId } = useParams();

    const edges = getDeploymentEdges(selectedDeployment);

    return (
        <NetworkEntityTabbedOverlay
            entityName={selectedDeployment.name}
            entityType={selectedDeployment.type}
        >
            <Tab title="Flows">
                <BinderTabs>
                    <Tab title="Network Flows">
                        <NetworkFlows
                            deploymentId={deploymentId}
                            edges={edges}
                            filterState={filterState}
                            onNavigateToEntity={onNavigateToEntity}
                            lastUpdatedTimestamp={lastUpdatedTimestamp}
                        />
                    </Tab>
                    <Tab title="Baseline Settings">
                        <BaselineSettings
                            selectedDeployment={selectedDeployment}
                            deploymentId={deploymentId}
                            filterState={filterState}
                        />
                    </Tab>
                </BinderTabs>
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
