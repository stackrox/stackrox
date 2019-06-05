import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { types as deploymentTypes } from 'reducers/deployments';
import { actions as pageActions } from 'reducers/network/page';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as graphActions } from 'reducers/network/graph';
import * as Icon from 'react-feather';
import isEmpty from 'lodash/isEmpty';

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
    if (
        !props.wizardOpen ||
        props.wizardStage !== wizardStages.details ||
        isEmpty(props.deployment)
    ) {
        return null;
    }

    const { deployment, selectedNode } = props;
    const { deployment: curDeployment } = deployment;
    const envGraphPanelTabs = [
        { text: 'Details' },
        { text: 'Network Policies' },
        { text: 'Network Flows' }
    ];
    const deploymentEdges = selectedNode.edges.filter(
        ({ data }) => data.destNodeNS && data.destNodeName && data.source !== data.target
    );

    function onDeploymentClick(id) {
        props.history.push(`/main/network/${id}`);
    }

    const content = props.isFetchingNode ? (
        <Loader />
    ) : (
        <Tabs headers={envGraphPanelTabs}>
            <TabContent>
                <div className="flex flex-1 flex-col h-full">
                    {curDeployment.id && <DeploymentDetails deployment={curDeployment} />}
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col h-full">
                    <NetworkPoliciesDetails />
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col h-full">
                    <DeploymentNetworkFlows
                        deploymentEdges={deploymentEdges}
                        onDeploymentClick={onDeploymentClick}
                    />
                </div>
            </TabContent>
        </Tabs>
    );

    function closeHandler() {
        const { onClose, networkGraphRef } = props;
        onClose();
        props.history.push('/main/network');
        if (networkGraphRef) props.networkGraphRef.setSelectedNode();
    }

    function onBackButtonClick() {
        const { setWizardStage, networkGraphRef } = props;
        setWizardStage(wizardStages.namespaceDetails);
        props.history.push('/main/network');
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
        <Panel leftButtons={leftButtons} header={curDeployment.name} onClose={closeHandler}>
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
    setSelectedNode: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired
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

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(Details)
);
