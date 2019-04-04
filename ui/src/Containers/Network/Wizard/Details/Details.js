import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { types as deploymentTypes } from 'reducers/deployments';
import { actions as pageActions } from 'reducers/network/page';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as graphActions } from 'reducers/network/graph';
import * as Icon from 'react-feather';

import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import PanelButton from 'Components/PanelButton';

import NetworkPoliciesDetails from './NetworkPoliciesDetails';
import DeploymentNetworkFlows from './DeploymentNetworkFlows';
import wizardStages from '../wizardStages';
import DeploymentDetails from '../../../Risk/DeploymentDetails';

function Details(props) {
    if (!props.wizardOpen || props.wizardStage !== wizardStages.details) {
        return null;
    }

    const { deployment, selectedNode } = props;
    const envGraphPanelTabs = [{ text: 'Details' }, { text: 'Network Policies' }];
    if (process.env.NODE_ENV === 'development') {
        envGraphPanelTabs.push({ text: 'Network Flows' });
    }
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
            {process.env.NODE_ENV === 'development' && (
                <TabContent>
                    <div className="flex flex-1 flex-col h-full">
                        <DeploymentNetworkFlows deploymentEdges={selectedNode.edges} />
                    </div>
                </TabContent>
            )}
        </Tabs>
    );

    function closeHandler() {
        const { onClose, networkGraphRef } = props;
        onClose();
        if (networkGraphRef) props.networkGraphRef.setSelectedNode();
    }

    function onBackButtonClick() {
        const { setWizardStage, networkGraphRef } = props;
        setWizardStage(wizardStages.namespaceDetails);
        if (networkGraphRef) {
            props.networkGraphRef.setSelectedNode();
            props.setSelectedNode(null);
        }
    }

    const leftButtons = props.selectedNamespace ? (
        <React.Fragment>
            <PanelButton
                icon={<Icon.ArrowLeft className="h-5 w-5" />}
                className="flex pl-3 text-center text-sm items-center"
                onClick={onBackButtonClick}
            />
        </React.Fragment>
    ) : null;

    return (
        <Panel leftButtons={leftButtons} header={deployment.name} onClose={closeHandler}>
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
    selectedNode: PropTypes.shape({}),
    selectedNamespace: PropTypes.shape({}),
    isFetchingNode: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func
    }),
    setWizardStage: PropTypes.func.isRequired,
    setSelectedNode: PropTypes.func.isRequired
};

Details.defaultProps = {
    networkGraphRef: null,
    selectedNode: null,
    selectedNamespace: null
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,

    networkPolicyGraph: selectors.getNetworkPolicyGraph,
    nodeUpdatesEpoch: selectors.getNodeUpdatesEpoch,
    deployment: selectors.getNodeDeployment,
    selectedNode: selectors.getSelectedNode,
    selectedNamespace: selectors.getSelectedNamespace,
    isFetchingNode: state => selectors.getLoadingStatus(state, deploymentTypes.FETCH_DEPLOYMENT),
    networkGraphRef: selectors.getNetworkGraphRef
});

const mapDispatchToProps = {
    onClose: pageActions.closeNetworkWizard,
    setWizardStage: wizardActions.setNetworkWizardStage,
    setSelectedNode: graphActions.setSelectedNode
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Details);
