import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as graphActions } from 'reducers/network/graph';

import Panel from 'Components/Panel';

import NetworkFlows from '../Details/NetworkFlows';
import wizardStages from '../wizardStages';

function ExternalDetails({
    history,
    networkGraphRef,
    onClose,
    wizardOpen,
    selectedNode,
    wizardStage,
}) {
    if (!wizardOpen || wizardStage !== wizardStages.externalDetails || !selectedNode) {
        return null;
    }

    function onNavigateToDeploymentById(deploymentId) {
        return function onNavigate() {
            history.push(`/main/network/${deploymentId}`);
        };
    }

    // TODO: redo this without Redux for External Entities
    const { edges, cidr, name } = selectedNode;

    function closeHandler() {
        onClose();
        history.push('/main/network');
        if (networkGraphRef) {
            networkGraphRef.setSelectedNode();
        }
    }

    const headerName = cidr ? `${name} | ${cidr}` : name;
    const panelId = cidr ? 'cidr-block-detail-panel' : 'external-entities-detail-panel';

    return (
        <Panel header={headerName} onClose={closeHandler} id={panelId}>
            <div className="flex flex-1 flex-col h-full">
                <NetworkFlows
                    edges={edges}
                    onNavigateToDeploymentById={onNavigateToDeploymentById}
                />
            </div>
        </Panel>
    );
}

ExternalDetails.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    wizardStage: PropTypes.string.isRequired,

    selectedNode: PropTypes.shape(),
    onClose: PropTypes.func.isRequired,
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getConfigObj: PropTypes.func,
        getNodeData: PropTypes.func,
    }),
    setSelectedNode: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

ExternalDetails.defaultProps = {
    networkGraphRef: null,
    selectedNode: null,
};

const mapStateToProps = createStructuredSelector({
    wizardOpen: selectors.getNetworkWizardOpen,
    wizardStage: selectors.getNetworkWizardStage,
    selectedNode: selectors.getSelectedNode,
    networkGraphRef: selectors.getNetworkGraphRef,
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setNetworkWizardStage,
    setSelectedNode: graphActions.setSelectedNode,
};

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(ExternalDetails));
