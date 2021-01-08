import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';

import useNavigateToDeployment from 'hooks/useNavigateToDeployment';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as graphActions } from 'reducers/network/graph';
import NetworkFlows from '../Details/NetworkFlows';

function ExternalDetailsOverlay({ selectedNode }): ReactElement {
    const onNavigateToDeploymentById = useNavigateToDeployment();

    const { edges, cidr, name } = selectedNode;
    const headerName = cidr ? `${name} | ${cidr}` : name;

    // TODO: generalize the layout wrapper in NetworkEntityTabbedOverlay.js so tabs are optional
    return (
        <div className="flex flex-1 flex-col text-sm max-h-minus-buttons">
            <div className="bg-primary-800 flex items-center m-2 min-w-108 p-3 rounded-lg shadow text-primary-100">
                <div className="flex flex-1 flex-col">
                    <div>{headerName}</div>
                    <div className="italic text-primary-200 text-xs capitalize">
                        Connected entities outside your cluster
                    </div>
                </div>
            </div>
            <div className="flex flex-1 m-2 pb-1 overflow-auto rounded bg-base-100">
                <NetworkFlows
                    edges={edges}
                    filterState={1}
                    onNavigateToDeploymentById={onNavigateToDeploymentById}
                />
            </div>
        </div>
    );
}

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

export default connect(mapStateToProps, mapDispatchToProps)(ExternalDetailsOverlay);
