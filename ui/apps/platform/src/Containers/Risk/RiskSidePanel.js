import React, { useEffect, useState } from 'react';
import PropTypes from 'prop-types';

import { fetchDeploymentWithRisk } from 'services/DeploymentsService';
import { fetchProcesses } from 'services/ProcessesService';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';

import RiskSidePanelContent from './RiskSidePanelContent';

function RiskSidePanel({ selectedDeploymentId, setSelectedDeploymentId }) {
    const [selectedDeployment, setSelectedDeployment] = useState(undefined);
    const [selectedProcesses, setSelectedProcesses] = useState(undefined);

    const [isFetching, setIsFetching] = useState(false);

    useEffect(() => {
        if (!selectedDeploymentId) {
            setSelectedDeployment(undefined);
            return;
        }

        setIsFetching(true);
        Promise.all([
            fetchDeploymentWithRisk(selectedDeploymentId),
            fetchProcesses(selectedDeploymentId),
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
    }, [selectedDeploymentId, setSelectedDeployment, setSelectedProcesses, setIsFetching]);

    function unselectDeployment() {
        setSelectedDeploymentId(undefined);
    }

    // Only render if we have image data to render.
    if (!selectedDeploymentId) {
        return null;
    }

    const header = !selectedDeployment ? 'Unknown Deployment' : selectedDeployment.deployment.name;

    /*
     * For border color compatible with background color of SidePanelAdjacentArea:
     */
    return (
        <PanelNew testid="panel">
            <PanelHead>
                <PanelTitle testid="panel-header" text={header} />
                <PanelHeadEnd>
                    <CloseButton
                        onClose={unselectDeployment}
                        className="border-base-400 border-l"
                    />
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <RiskSidePanelContent
                    isFetching={isFetching}
                    selectedDeployment={selectedDeployment ? selectedDeployment.deployment : null}
                    deploymentRisk={selectedDeployment ? selectedDeployment.risk : null}
                    processGroup={selectedProcesses}
                />
            </PanelBody>
        </PanelNew>
    );
}

RiskSidePanel.propTypes = {
    selectedDeploymentId: PropTypes.string,
    setSelectedDeploymentId: PropTypes.func.isRequired,
};

RiskSidePanel.defaultProps = {
    selectedDeploymentId: undefined,
};

export default RiskSidePanel;
