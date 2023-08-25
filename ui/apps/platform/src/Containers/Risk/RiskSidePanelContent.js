import React from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { Alert } from '@patternfly/react-core';

import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import Loader from 'Components/Loader';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import usePermissions from 'hooks/usePermissions';

import { getURLLinkToDeployment } from 'Containers/NetworkGraph/utils/networkGraphURLUtils';
import RiskDetails from './RiskDetails';
import DeploymentDetails from './DeploymentDetails';
import ProcessDetails from './Process/Details';

function RiskSidePanelContent({ isFetching, selectedDeployment, deploymentRisk, processGroup }) {
    const isRouteEnabled = useIsRouteEnabled();
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAlert = hasReadAccess('Alert');
    const isRouteEnabledForNetworkGraph = isRouteEnabled('network-graph');

    if (isFetching) {
        return <Loader />;
    }

    if (!selectedDeployment) {
        return (
            <div className="h-full flex-1 bg-base-200 border-r border-l border-b border-base-400 p-3">
                <Alert variant="warning" isInline title="Deployment not found">
                    The selected deployment may have been removed.
                </Alert>
            </div>
        );
    }

    const riskPanelTabs = [{ text: 'Risk Indicators' }, { text: 'Deployment Details' }];
    if (hasReadAccessForAlert) {
        riskPanelTabs.push({ text: 'Process Discovery' });
    }

    const networkGraphLink = getURLLinkToDeployment({
        cluster: selectedDeployment.clusterName,
        namespace: selectedDeployment.namespace,
        deploymentId: selectedDeployment.id,
    });

    return (
        <Tabs headers={riskPanelTabs}>
            <Tab>
                <div className="flex flex-col pb-5">
                    {isRouteEnabledForNetworkGraph && (
                        <Link
                            className="btn btn-base h-10 no-underline mt-4 ml-3 mr-3"
                            to={networkGraphLink}
                            data-testid="view-deployments-in-network-graph-button"
                        >
                            View Deployment in Network Graph
                        </Link>
                    )}
                    {!deploymentRisk ? (
                        <Alert variant="warning" isInline title="Risk not found">
                            Risk for selected deployment may not have been processed.
                        </Alert>
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
            {hasReadAccessForAlert && (
                <Tab>
                    <div className="flex flex-1 flex-col relative">
                        {!processGroup ||
                        !processGroup.groups ||
                        processGroup.groups.length === 0 ? (
                            <Alert variant="warning" isInline title="No processes discovered">
                                <p>
                                    The selected deployment may not have running pods, or Collector
                                    may not be running in your cluster.
                                </p>
                                <p>It is recommended to check the logs for more information.</p>
                            </Alert>
                        ) : (
                            <ProcessDetails
                                processGroup={processGroup}
                                deploymentId={selectedDeployment.id}
                            />
                        )}
                    </div>
                </Tab>
            )}
        </Tabs>
    );
}

RiskSidePanelContent.propTypes = {
    isFetching: PropTypes.bool.isRequired,
    selectedDeployment: PropTypes.shape({
        id: PropTypes.string.isRequired,
        clusterName: PropTypes.string.isRequired,
        namespace: PropTypes.string.isRequired,
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
