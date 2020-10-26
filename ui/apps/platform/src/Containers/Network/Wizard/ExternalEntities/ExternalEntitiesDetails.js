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

function ExternalEntitiesFlows({ history, networkGraphRef, onClose, wizardOpen, wizardStage }) {
    if (!wizardOpen || wizardStage !== wizardStages.externalEntitiesFlows) {
        return null;
    }

    function onNavigateToDeploymentById(deploymentId) {
        return function onNavigate() {
            history.push(`/main/network/${deploymentId}`);
        };
    }

    // TODO: redo this without Redux for External Entities
    const deploymentEdges = [];

    function closeHandler() {
        onClose();
        history.push('/main/network');
        if (networkGraphRef) {
            networkGraphRef.setSelectedNode();
        }
    }

    return (
        <Panel header="External Entities" onClose={closeHandler} id="network-details-panel">
            <div className="flex flex-1 flex-col h-full">
                <NetworkFlows
                    deploymentEdges={deploymentEdges}
                    onNavigateToDeploymentById={onNavigateToDeploymentById}
                />
            </div>
        </Panel>
    );
}

ExternalEntitiesFlows.propTypes = {
    wizardOpen: PropTypes.bool.isRequired,
    wizardStage: PropTypes.string.isRequired,

    selectedNode: PropTypes.shape({
        outEdges: PropTypes.arrayOf(PropTypes.shape({})),
        id: PropTypes.string.isRequired,
    }),
    onClose: PropTypes.func.isRequired,
    networkGraphRef: PropTypes.shape({
        setSelectedNode: PropTypes.func,
        getConfigObj: PropTypes.func,
        getNodeData: PropTypes.func,
    }),
    setSelectedNode: PropTypes.func.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
};

ExternalEntitiesFlows.defaultProps = {
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

export default withRouter(connect(mapStateToProps, mapDispatchToProps)(ExternalEntitiesFlows));
