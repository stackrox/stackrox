import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';

import Tabs from 'Components/Tabs';
import Loader from 'Components/Loader';
import TabContent from 'Components/TabContent';
import Message from 'Components/Message';

import RiskDetails from './RiskDetails';
import DeploymentDetails from './DeploymentDetails';
import ProcessDetails from './Process/Details';

const riskErrMsg = `Risk not found. Risk for selected deployment may not have been processed.`;
const deploymentErrMsg = `Deployment not found. The selected deployment may have been removed.`;
const processErrMsg = `No processes discovered. The selected deployment may not have running pods,
    or the StackRox collector may not be running in your cluster.
    It is recommended to check the logs for more information.`;

const RiskSidePanelErrorContent = ({ message }) => {
    return (
        <div className="h-full flex-1 bg-base-200 border-r border-l border-b border-base-400 p-3">
            <Message message={message} type="error" />
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
            <TabContent>
                <div className="flex flex-col pb-5">
                    <Link
                        className="btn btn-base h-10 no-underline mt-4 ml-3 mr-3"
                        to={`/main/network/${selectedDeployment.id}`}
                        data-testid="network-node-link"
                    >
                        View Deployment in Network Graph
                    </Link>
                    {!deploymentRisk ? (
                        <RiskSidePanelErrorContent message={riskErrMsg} />
                    ) : (
                        <RiskDetails risk={deploymentRisk} />
                    )}
                </div>
            </TabContent>
            <TabContent>
                <div className="flex flex-1 flex-col relative">
                    <div className="absolute w-full">
                        <DeploymentDetails deployment={selectedDeployment} />
                    </div>
                </div>
            </TabContent>
            <TabContent>
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
            </TabContent>
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
