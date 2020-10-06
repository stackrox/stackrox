import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import * as Icon from 'react-feather';
import isEmpty from 'lodash/isEmpty';

import { types as deploymentTypes } from 'reducers/deployments';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as graphActions } from 'reducers/network/graph';
import Panel from 'Components/Panel';
import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import PanelButton from 'Components/PanelButton';
import NetworkPoliciesDetails from './NetworkPoliciesDetails';
import NetworkFlows from './NetworkFlows';
import wizardStages from '../wizardStages';
import DeploymentDetails from '../../../Risk/DeploymentDetails';

function Details({
    deployment,
    selectedNode,
    networkGraphRef,
    history,
    isFetchingNode,
    onClose,
    selectedNamespace,
    setWizardStage,
    setSelectedNode,
}) {
    if (isEmpty(deployment)) {
        return null;
    }

    const { deployment: curDeployment } = deployment;
    const envGraphPanelTabs = [
        { text: 'Network Flows' },
        { text: 'Details' },
        { text: 'Network Policies' },
    ];

    const deploymentEdges = selectedNode.edges.filter(
        ({ data }) => data.destNodeNamespace && data.destNodeName && data.source !== data.target
    );

    function onNavigateToDeploymentById(deploymentId) {
        return function onNavigate() {
            history.push(`/main/network/${deploymentId}`);
        };
    }

    const content = isFetchingNode ? (
        <Loader />
    ) : (
        <Tabs headers={envGraphPanelTabs}>
            <TabContent>
                <div className="flex flex-1 flex-col h-full">
                    <NetworkFlows
                        deploymentEdges={deploymentEdges}
                        onNavigateToDeploymentById={onNavigateToDeploymentById}
                    />
                </div>
            </TabContent>
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
        </Tabs>
    );

    function closeHandler() {
        onClose();
        history.push('/main/network');
        if (networkGraphRef) {
            networkGraphRef.setSelectedNode();
        }
    }

    function onBackButtonClick() {
        setWizardStage(wizardStages.namespaceDetails);
        history.push('/main/network');
        if (networkGraphRef) {
            networkGraphRef.setSelectedNode();
            setSelectedNode(null);
        }
    }

    const leftButtons = selectedNamespace ? (
        <PanelButton
            icon={<Icon.ArrowLeft className="h-5 w-5" />}
            className="flex pl-3 text-center text-sm items-center"
            onClick={onBackButtonClick}
            tooltip="Back"
        />
    ) : null;

    return (
        <Panel
            leftButtons={leftButtons}
            header={curDeployment.name}
            onClose={closeHandler}
            id="network-details-panel"
        >
            {content}
        </Panel>
    );
}

Details.propTypes = {
    deployment: PropTypes.shape({
        name: PropTypes.string,
        deployment: PropTypes.shape({}),
    }).isRequired,
    selectedNode: PropTypes.shape({
        edges: PropTypes.arrayOf(PropTypes.shape({})),
        id: PropTypes.string.isRequired,
    }),
    selectedNamespace: PropTypes.shape({}),
    isFetchingNode: PropTypes.bool.isRequired,
    onClose: PropTypes.func.isRequired,
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getConfigObj: PropTypes.func,
        getNodeData: PropTypes.func,
    }),
    setWizardStage: PropTypes.func.isRequired,
    setSelectedNode: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

Details.defaultProps = {
    networkGraphRef: null,
    selectedNode: null,
    selectedNamespace: null,
};

const mapStateToProps = createStructuredSelector({
    deployment: selectors.getNodeDeployment,
    selectedNode: selectors.getSelectedNode,
    selectedNamespace: selectors.getSelectedNamespace,
    isFetchingNode: (state) => selectors.getLoadingStatus(state, deploymentTypes.FETCH_DEPLOYMENT),
    networkGraphRef: selectors.getNetworkGraphRef,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setNetworkWizardStage,
    setSelectedNode: graphActions.setSelectedNode,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(Details));
