import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { fetchDeployment } from 'services/DeploymentsService';
import fetchRisk from 'services/RisksService';
import { fetchProcesses } from 'services/ProcessesService';

import Panel from 'Components/Panel';
import RiskSidePanelContent from './RiskSidePanelContent';

function RiskSidePanel({ selectedDeploymentId, setSelectedDeploymentId }) {
    const [selectedDeployment, setSelectedDeployment] = useState(undefined);
    const [selectedProcesses, setSelectedProcesses] = useState(undefined);
    const [selectedRisk, setSelectedRisk] = useState(undefined);

    const [isFetching, setIsFetching] = useState(false);

    useEffect(
        () => {
            if (!selectedDeploymentId) {
                setSelectedDeployment(undefined);
                return;
            }

            setIsFetching(true);
            Promise.all([
                fetchDeployment(selectedDeploymentId),
                fetchRisk(selectedDeploymentId, 'deployment'),
                fetchProcesses(selectedDeploymentId)
            ]).then(
                ([deployment, risk, processes]) => {
                    setSelectedDeployment(deployment);
                    setSelectedRisk(risk);
                    setSelectedProcesses(processes.response);
                    setIsFetching(false);
                },
                () => {
                    setSelectedDeployment(undefined);
                    setSelectedRisk(undefined);
                    setSelectedProcesses(undefined);
                    setIsFetching(false);
                }
            );
        },
        [
            selectedDeploymentId,
            setSelectedDeployment,
            setSelectedRisk,
            setSelectedProcesses,
            setIsFetching
        ]
    );

    function unselectDeployment() {
        setSelectedDeploymentId(undefined);
    }

    // Only render if we have image data to render.
    if (!selectedDeploymentId || !selectedDeployment || !selectedRisk || !selectedProcesses)
        return null;

    return (
        <Panel
            header={selectedDeployment.name}
            className="bg-primary-200 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-72 md:relative"
            onClose={unselectDeployment}
        >
            <RiskSidePanelContent
                isFetching={isFetching}
                selectedDeployment={selectedDeployment}
                deploymentRisk={selectedRisk}
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
