import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { types as deploymentTypes } from 'reducers/deployments';
import { actions as pageActions } from 'reducers/network/page';
import { selectors } from 'reducers';

import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';

import NetworkPoliciesDetails from './NetworkPoliciesDetails';
import wizardStages from '../wizardStages';
import DeploymentDetails from '../../../Risk/DeploymentDetails';

function Details(props) {
    if (!props.wizardOpen || props.wizardStage !== wizardStages.details) {
        return null;
    }

    const { deployment } = props;
    const envGraphPanelTabs = [{ text: 'Deployment Details' }, { text: 'Network Policies' }];
    const content = props.isFetchingNode ? (
        <Loader />
    ) : (
        <Tabs headers={envGraphPanelTabs}>
            <TabContent>
                <div className="flex flex-1 flex-col h-full">
                    {deployment.id && <DeploymentDetails deployment={deployment} />}
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col h-full">
                    <NetworkPoliciesDetails />
                </div>
            </TabContent>
        </Tabs>
    );

    function closeHandler() {
        const { onClose, networkGraphRef } = props;
        onClose();
        if (networkGraphRef) props.networkGraphRef.setSelectedNode();
    }

    return (
        <Panel header={deployment.name} onClose={closeHandler}>
            {content}
        </Panel>
    );
}

Details.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    wizardStage: PropTypes.string.isRequired,

    deployment: PropTypes.shape({
        name: PropTypes.string
    }).isRequired,

    isFetchingNode: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func
    })
};

Details.defaultProps = {
    networkGraphRef: null
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,

    networkPolicyGraph: selectors.getNetworkPolicyGraph,
    nodeUpdatesEpoch: selectors.getNodeUpdatesEpoch,
    deployment: selectors.getNodeDeployment,
    isFetchingNode: state => selectors.getLoadingStatus(state, deploymentTypes.FETCH_DEPLOYMENT),
    networkGraphRef: selectors.getNetworkGraphRef
});

const mapDispatchToProps = {
    onClose: pageActions.closeNetworkWizard
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Details);
