import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { Message } from '@stackrox/ui-components';

import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import Loader from 'Components/Loader';

import RiskDetails from './RiskDetails';
import DeploymentDetails from './DeploymentDetails';
import ProcessDetails from './Process/Details';

const riskErrMsg = `Risk not found. Risk for selected deployment may not have been processed.`;
const deploymentErrMsg = `Deployment not found. The selected deployment may have been removed.`;
const processErrMsg = `No processes discovered. The selected deployment may not have running pods,
    or Collector may not be running in your cluster.
    It is recommended to check the logs for more information.`;

const RiskSidePanelErrorContent = ({ message }) => {
    return (
        <div className="h-full flex-1 bg-base-200 border-r border-l border-b border-base-400 p-3">
            <Message type="error">{message}</Message>
        </div>
    );
};

function RiskSidePanelContent({ isFetching, selectedDeployment, deploymentRisk, processGroup }) {
    if (isFetching) {
        return <Loader />;
    }

    if (!selectedDeployment) {
        return <RiskSidePanelErrorContent message={deploymentErrMsg} />;
    }

    const riskPanelTabs = [
        { text: 'Risk Indicators' },
        { text: 'Deployment Details' },
        { text: 'Process Discovery' },
    ];
    return (
        <Tabs headers={riskPanelTabs}>
            <Tab>
                <div className="flex flex-col pb-5">
                    <Link
                        className="btn btn-base h-10 no-underline mt-4 ml-3 mr-3"
                        to={`/main/network/${selectedDeployment.id}`}
                        data-testid="view-deployments-in-network-graph-button"
                    >
                        View Deployment in Network Graph
                    </Link>
                    {!deploymentRisk ? (
                        <RiskSidePanelErrorContent message={riskErrMsg} />
                    ) : (
                        <RiskDetails risk={deploymentRisk} />
                    )}
                </div>
            </Tab>
            <Tab>
                <div className="flex flex-1 flex-col relative">
                    <div className="absolute w-full">
                        <DeploymentDetails deployment={selectedDeployment} />
                    </div>
                </div>
            </Tab>
            <Tab>
                <div className="flex flex-1 flex-col relative">
                    {!processGroup || !processGroup.groups || processGroup.groups.length === 0 ? (
                        <RiskSidePanelErrorContent message={processErrMsg} />
                    ) : (
                        <ProcessDetails
                            processGroup={processGroup}
                            deploymentId={selectedDeployment.id}
                        />
                    )}
                </div>
            </Tab>
        </Tabs>
    );
}

RiskSidePanelErrorContent.propTypes = {
    message: PropTypes.string.isRequired,
};

RiskSidePanelContent.propTypes = {
    isFetching: PropTypes.bool.isRequired,
    selectedDeployment: PropTypes.shape({
        id: PropTypes.string.isRequired,
    }),
    deploymentRisk: PropTypes.shape({}),
    processGroup: PropTypes.shape({
        groups: PropTypes.arrayOf(PropTypes.object),
    }),
};

RiskSidePanelContent.defaultProps = {
    selectedDeployment: undefined,
    deploymentRisk: undefined,
    processGroup: undefined,
};

export default RiskSidePanelContent;
