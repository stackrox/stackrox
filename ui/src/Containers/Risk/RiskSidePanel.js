import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { fetchDeploymentWithRisk } from 'services/DeploymentsService';
import { fetchProcesses } from 'services/ProcessesService';

import Panel from 'Components/Panel';
import RiskSidePanelContent from './RiskSidePanelContent';

function RiskSidePanel({ selectedDeploymentId, setSelectedDeploymentId }) {
    const [selectedDeployment, setSelectedDeployment] = useState(undefined);
    const [selectedProcesses, setSelectedProcesses] = useState(undefined);

    const [isFetching, setIsFetching] = useState(false);

    useEffect(
        () => {
            if (!selectedDeploymentId) {
                setSelectedDeployment(undefined);
                return;
            }

            setIsFetching(true);
            Promise.all([
                fetchDeploymentWithRisk(selectedDeploymentId),
                fetchProcesses(selectedDeploymentId)
            ]).then(
                ([deploymentWithRisk, processes]) => {
                    setSelectedDeployment(deploymentWithRisk);
                    setSelectedProcesses(processes.response);
                    setIsFetching(false);
                },
                () => {
                    setSelectedDeployment(undefined);
                    setSelectedProcesses(undefined);
                    setIsFetching(false);
                }
            );
        },
        [selectedDeploymentId, setSelectedDeployment, setSelectedProcesses, setIsFetching]
    );

    function unselectDeployment() {
        setSelectedDeploymentId(undefined);
    }

    // Only render if we have image data to render.
    if (!selectedDeploymentId) return null;
    return (
        <Panel
            header={!selectedDeployment ? 'Unknown Deployment' : selectedDeployment.deployment.name}
            className="bg-primary-200 w-full h-full absolute right-0 top-0 md:w-1/2 min-w-72 md:relative z-0 bg-base-100"
            onClose={unselectDeployment}
        >
            <RiskSidePanelContent
                isFetching={isFetching}
                selectedDeployment={!selectedDeployment ? null : selectedDeployment.deployment}
                deploymentRisk={!selectedDeployment ? null : selectedDeployment.risk}
                processGroup={selectedProcesses}
            />
        </Panel>
    );
}

RiskSidePanel.propTypes = {
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired
};

RiskSidePanel.defaultProps = {
    selectedDeploymentId: undefined
};

export default RiskSidePanel;
