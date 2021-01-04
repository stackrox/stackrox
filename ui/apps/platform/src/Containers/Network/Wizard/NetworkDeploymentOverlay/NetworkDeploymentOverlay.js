import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { useParams, useHistory } from 'react-router-dom';

import { selectors } from 'reducers';
import { nodeTypes } from 'constants/networkGraph';

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

function useNavigateToDeployment() {
    const history = useHistory();
    return function onNavigateToDeploymentById(deploymentId, type) {
        return function onNavigate() {
            if (type === 'external' || type === 'cidr') {
                history.push(`/main/network/${deploymentId}/${type}`);
                return;
            }
            history.push(`/main/network/${deploymentId}`);
        };
    };
}

function NetworkDeploymentOverlay({ selectedDeployment, filterState }) {
    const onNavigateToDeploymentById = useNavigateToDeployment();
    const { deploymentId } = useParams();

    const edges = getDeploymentEdges(selectedDeployment);

    return (
        <div className="flex flex-1 flex-col text-sm max-h-minus-buttons">
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
                                onNavigateToDeploymentById={onNavigateToDeploymentById}
                            />
                        </Tab>
                        <Tab title="Baseline Settings">
                            <BaselineSettings />
                        </Tab>
                    </BinderTabs>
                </Tab>
                <Tab title="Policies">
                    <NetworkPoliciesDetail policyIds={selectedDeployment.policyIds} />
                </Tab>
                <Tab title="Details">
                    <DeploymentDetails deploymentId={deploymentId} />
                </Tab>
            </NetworkEntityTabbedOverlay>
        </div>
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
};

const mapStateToProps = createStructuredSelector({
    selectedDeployment: selectors.getSelectedNode,
    filterState: selectors.getNetworkGraphFilterMode,
});

export default connect(mapStateToProps, null)(NetworkDeploymentOverlay);
