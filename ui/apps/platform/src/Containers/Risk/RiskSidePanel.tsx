import { useEffect, useState } from 'react';
import { Alert, Bullseye, Spinner } from '@patternfly/react-core';

import { fetchDeploymentWithRisk } from 'services/DeploymentsService';
import type { DeploymentWithRisk } from 'services/DeploymentsService';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';

import RiskSidePanelTabs from './RiskSidePanelTabs';

export type RiskSidePanelProps = {
    selectedDeploymentId: string;
    setSelectedDeploymentId: (string) => void;
};

function RiskSidePanel({ selectedDeploymentId, setSelectedDeploymentId }: RiskSidePanelProps) {
    const [selectedDeployment, setSelectedDeployment] = useState<DeploymentWithRisk | null>(null);
    const [isFetching, setIsFetching] = useState(false);

    useEffect(() => {
        setIsFetching(true);
        fetchDeploymentWithRisk(selectedDeploymentId)
            .then((deploymentWithRisk) => {
                setSelectedDeployment(deploymentWithRisk);
            })
            .catch(() => {
                setSelectedDeployment(null);
            })
            .finally(() => {
                setIsFetching(false);
            });
    }, [selectedDeploymentId, setIsFetching]);

    function unselectDeployment() {
        setSelectedDeploymentId(undefined);
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
                {isFetching ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : !selectedDeployment?.deployment ? (
                    <Alert variant="warning" isInline title="Deployment not found" component="p">
                        The selected deployment may have been removed.
                    </Alert>
                ) : (
                    <RiskSidePanelTabs
                        deployment={selectedDeployment.deployment}
                        risk={selectedDeployment.risk}
                    />
                )}
            </PanelBody>
        </PanelNew>
    );
}

export default RiskSidePanel;
